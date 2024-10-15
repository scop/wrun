// Copyright 2024 Ville Skyttä
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"crypto"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"slices"
	"strings"

	pep440 "github.com/aquasecurity/go-pep440-version"
	"github.com/spf13/cobra"

	"github.com/scop/wrun/internal/pypi"
)

func generateArbitraryPyPIProjectCommand(w *Wrun) *cobra.Command {
	var tool, release string
	genCmd := &cobra.Command{
		Use:   "pypi PROJECT",
		Short: "generate wrun command line arguments for tool in PyPI project wrapper wheel",
		Example: strings.TrimSpace("" +
			w.ProgName + " generate pypi committed\n" +
			w.ProgName + " generate pypi ruff\n" +
			w.ProgName + " generate pypi typos\n" +
			""),
		ValidArgsFunction: cobra.NoFileCompletions,
		Args:              cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			if tool == "" {
				tool = args[0] // Default tool = project name
			}
			if err := runGeneratePyPIProject(w, args[0], tool, release); err != nil {
				w.LogError("%s", err)
				os.Exit(1)
			}
		},
	}
	genCmd.Flags().StringVarP(&tool, "tool", "T", "", "tool name to search within archive, defaults to project name")
	if err := genCmd.RegisterFlagCompletionFunc("tool", cobra.NoFileCompletions); err != nil {
		w.LogBug("register --tool completion: %s", err)
	}
	genCmd.Flags().StringVarP(&tool, "release", "r", "", "project release version, defaults to automatically selected")
	if err := genCmd.RegisterFlagCompletionFunc("release", pypiVersionCompleter(w)); err != nil {
		w.LogBug("register --release completion: %s", err)
	}

	return genCmd
}

func pypiVersionCompleter(w *Wrun) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		p, err := getPyPIProject(w, args[0])
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		versions := p.ValidVersions()
		ret := make([]string, 0, len(versions))
		for _, v := range versions {
			vs := v.String()
			if strings.HasPrefix(vs, toComplete) {
				ret = append(ret, vs)
			}
		}

		return ret, cobra.ShellCompDirectiveNoFileComp
	}
}

func preferredVersion(p pypi.SimpleProject) string {
	if len(p.Files) == 0 {
		return ""
	}
	versions := p.ValidVersions()
	versionFound := false
	var v pep440.Version
	for _, v = range versions {
		// TODO check that this version has binary distributions
		if !v.IsPreRelease() {
			break
		}
	}
	if !versionFound {
		v = versions[0]
	}

	return v.String()
}

func getPyPIProject(w *Wrun, project string) (*pypi.SimpleProject, error) {
	url := fmt.Sprintf("https://pypi.org/simple/%s/", url.PathEscape(project))
	resp, err := w.HTTPGet(url, "Accept:application/vnd.pypi.simple.v1+json")
	if err != nil {
		return nil, fmt.Errorf("get %s versions: %w", project, err)
	}
	var p pypi.SimpleProject
	err = json.NewDecoder(resp.Body).Decode(&p)
	if cErr := resp.Body.Close(); cErr != nil {
		w.LogWarn("close %s body: %v", url, cErr)
	}
	if err != nil {
		return nil, fmt.Errorf("unmarshal project from %s: %w", url, err)
	}

	return &p, nil
}

func runGeneratePyPIProject(w *Wrun, project, tool, version string) error {
	p, err := getPyPIProject(w, project)
	if err != nil {
		return err
	}

	if version == "" {
		version = preferredVersion(*p)
	} else {
		version = strings.TrimPrefix(version, "v")
	}

	osArchFiles, otherFiles := p.PreferredOsArchSimpleFiles(version)
	for _, file := range otherFiles {
		w.LogWarn("no matching pattern for %q, ignoring", file.Filename)
	}
	exePaths := make(map[string]string, len(osArchFiles))

	// Process os/arch assets sorted by os/arch for stable output
	osArchs := make([]string, 0, len(osArchFiles))
	for osArch := range osArchFiles {
		osArchs = append(osArchs, osArch)
	}
	slices.Sort(osArchs)

	hshType := crypto.SHA256
	hsh := hshType.New()
	for _, osArch := range osArchs {
		pf := osArchFiles[osArch]
		if pf.URL == "" {
			w.LogWarn("missing URL for %q, ignoring", pf.Filename)

			continue
		}
		if pf.Hashes.SHA256 == "" {
			w.LogWarn("missing SHA-256 hash for %q, ignoring", pf.Filename)

			continue
		}
		expectedDigest, err := hex.DecodeString(pf.Hashes.SHA256)
		if err != nil {
			return fmt.Errorf("decode hex digest: %w", err)
		}

		resp, err := w.HTTPGet(pf.URL)
		if err != nil {
			return err
		}

		tmpf, cleanupTempfile, err := w.SetUpTempfile(pf.URL, "")
		if err != nil {
			return fmt.Errorf("set up tempfile: %w", err)
		}
		if err = w.Download(resp, tmpf, hsh, expectedDigest); err != nil {
			cleanupTempfile()

			return fmt.Errorf("download: %w", err)
		}

		toolExe := tool
		if strings.HasPrefix(osArch, "windows/") {
			toolExe += ".exe"
		}
		exePath, err := findToolInArchive(tmpf.Name(), toolExe)
		cleanupTempfile()
		if err != nil {
			if !strings.Contains(err.Error(), "format unrecognized by filename") { // No better way as of archiver 3.5.1
				w.LogError("find tool in archive: %v", err)
			}
		} else {
			exePaths[osArch] = exePath
		}
		hsh.Reset()

		fmt.Printf("--url=%s=%s#sha256-%s\n", osArch, pf.URL, pf.Hashes.SHA256)
	}
	for _, ep := range generateExePathArgs(exePaths) {
		fmt.Printf("--archive-exe-path=%s\n", ep)
	}

	return nil
}
