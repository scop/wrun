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
	"github.com/scop/wrun/internal/pypi"
	"github.com/spf13/cobra"
)

func generateArbitraryPyPIProjectCommand(w *Wrun) *cobra.Command {
	genCmd := &cobra.Command{
		Use:   "pypi PROJECT [TOOL [VERSION]]",
		Short: "generate wrun command line arguments for a PyPI project",
		Args:  cobra.RangeArgs(1, 3),
		Run: func(_ *cobra.Command, args []string) {
			if len(args) == 1 { // Default tool = project
				args = append(args, args[0])
			}
			if err := runGeneratePyPIProject(w, args[0], args[1], args[2:]); err != nil {
				w.LogError("%s", err)
				os.Exit(1)
			}
		},
	}

	return genCmd
}

func generatePyPIProjectCommand(w *Wrun, project, tool string) *cobra.Command {
	genCmd := &cobra.Command{
		Use:   tool + " [VERSION]",
		Short: "generate wrun command line arguments for " + tool,
		Args:  cobra.MaximumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			if err := runGeneratePyPIProject(w, project, tool, args); err != nil {
				w.LogError("%s", err)
				os.Exit(1)
			}
		},
	}

	return genCmd
}

func preferredVersion(p pypi.SimpleProject) string {
	if len(p.Files) == 0 {
		return ""
	}
	versions := p.Versions()
	versionFound := false
	var v pep440.Version
	for _, v = range versions {
		// TODO check for yanked files with this version, skip if there are any ... or if all are?
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

func runGeneratePyPIProject(w *Wrun, project, tool string, args []string) error {
	url := fmt.Sprintf("https://pypi.org/simple/%s/", url.PathEscape(project))
	resp, err := w.HTTPGet(url, "Accept:application/vnd.pypi.simple.v1+json")
	if err != nil {
		return fmt.Errorf("get %s versions: %w", project, err)
	}
	var p pypi.SimpleProject
	err = json.NewDecoder(resp.Body).Decode(&p)
	if cErr := resp.Body.Close(); cErr != nil {
		w.LogWarn("close %s body: %v", url, cErr)
	}
	if err != nil {
		return fmt.Errorf("unmarshal project from %s: %w", url, err)
	}

	var version string
	if len(args) != 0 {
		version = strings.TrimPrefix(args[0], "v")
	} else {
		version = preferredVersion(p)
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

		// TODO skip download if skip-verify given
		if resp, err = w.HTTPGet(pf.URL); err != nil {
			return err
		}

		tmpf, cleanupTempfile, err := w.SetUpTempfile(pf.URL, "")
		if err != nil {
			return fmt.Errorf("set up tempfile: %w", err) // TODO think this through
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
				w.LogError("find tool in archive: %v", err) // TODO think this through, continue or fail?
			}
		} else {
			exePaths[osArch] = exePath
		}
		hsh.Reset()

		fmt.Printf("--url %s=%s#sha256-%s\n", osArch, pf.URL, pf.Hashes.SHA256)
	}
	for _, ep := range generateExePathArgs(exePaths) {
		fmt.Printf("--archive-exe-path %s\n", ep)
	}

	return nil
}
