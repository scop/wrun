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

	"github.com/scop/wrun/internal/pypi"
	"github.com/spf13/cobra"
)

func generateArbitraryPyPIProjectCommand(w *Wrun) *cobra.Command {
	genCmd := &cobra.Command{
		Use:   "pypi PROJECT TOOL [VERSION]",
		Short: "generate wrun command line arguments for a PyPI project",
		Args:  cobra.RangeArgs(2, 3),
		Run: func(_ *cobra.Command, args []string) {
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
		// TODO handle no versions
		vs := p.Versions()
		version = vs[0]
	}

	osArchFiles, fileMisses := p.PreferredOsArchSimpleFiles(version)
	for _, file := range fileMisses {
		w.LogWarn("no matching pattern for %q, ignoring", file.Filename)
	}
	exePaths := make([]string, 0, len(osArchFiles))

	// Process os/arch assets sorted by os/arch for stable output
	osArchs := make([]string, 0, len(osArchFiles))
	for osArch := range osArchFiles {
		osArchs = append(osArchs, osArch)
	}
	slices.Sort(osArchs)

	// TODO generate exe paths by locating the tool arg within archives, simplify like for "github" things

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
		if err = w.Download(resp, nil, hsh, expectedDigest); err != nil {
			return fmt.Errorf("download: %w", err)
		}
		hsh.Reset()

		fmt.Printf("--url %s=%s#sha256-%s\n", osArch, pf.URL, pf.Hashes.SHA256)
		ext := ""
		if strings.HasPrefix(osArch, "windows/") {
			ext = ".exe"
		}
		exePaths = append(exePaths, fmt.Sprintf("--archive-exe-path %s=%s-%s.data/scripts/%s%s", osArch, project, version, project, ext))
	}
	for _, ep := range exePaths {
		fmt.Println(ep)
	}

	return nil
}
