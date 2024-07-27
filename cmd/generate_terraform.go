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
	"bytes"
	"crypto"
	"fmt"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	util "github.com/scop/wrun/internal"
	"github.com/scop/wrun/internal/files"
)

func generateTerraformCommand(w *Wrun) *cobra.Command {
	genCmd := &cobra.Command{
		Use:   "terraform [VERSION]",
		Short: "generate wrun command line arguments for terraform",
		Args:  cobra.MaximumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			if err := runGenerateTerraform(w, args); err != nil {
				w.LogError("%s", err)
				os.Exit(1)
			}
		},
	}

	return genCmd
}

func runGenerateTerraform(w *Wrun, args []string) error {
	var version string
	if len(args) != 0 {
		version = args[0]
	} else {
		owner, project := "hashicorp", "terraform"
		rels, err := releasesFromGitHubAPI(w, owner, project)
		if err != nil {
			return fmt.Errorf("get %s/%s releases: %w", owner, project, err)
		}
		version = preferredRelease(rels).TagName
	}
	version = strings.TrimPrefix(version, "v")

	baseURL := "https://releases.hashicorp.com/terraform/" + url.PathEscape(version)

	var checksums util.Checksums
	var buf bytes.Buffer
	csURL := baseURL + "/terraform_" + url.PathEscape(version) + "_SHA256SUMS"
	if resp, err := w.HTTPGet(csURL); err != nil {
		return err
	} else if err := w.Download(resp, &buf, nil, nil); err != nil {
		return fmt.Errorf("download: %w", err)
	}
	if err := checksums.UnmarshalText(buf.Bytes()); err != nil {
		w.LogWarn("unmarshal checksums from %q: %v", csURL, err)
	}

	fileEntries := make(map[string][]util.ChecksumEntry, len(checksums.Entries))
	for _, ce := range checksums.Entries {
		if ces, found := fileEntries[ce.Filename]; found {
			fileEntries[ce.Filename] = append(ces, ce)
		} else {
			fileEntries[ce.Filename] = []util.ChecksumEntry{ce}
		}
	}
	osArchEntries, _, otherEntries := files.Categorize(fileEntries, nil)
	for _, ce := range otherEntries {
		w.LogWarn("no matching pattern for %q, ignoring", ce)
	}

	// Process os/arch entries sorted by os/arch for stable output between runs
	osArchs := make([]string, 0, len(osArchEntries))
	for osArch := range osArchEntries {
		osArchs = append(osArchs, osArch)
	}
	slices.Sort(osArchs)

	exePaths := make(map[string]string, len(osArchs))

	hn, err := util.HashByName(util.HashName(crypto.SHA256))
	if err != nil {
		return err
	}
	hsh := hn.New()

	const tool = "terraform"
	for _, osArch := range osArchs {
		var toolExe string
		if strings.HasPrefix(osArch, "windows/") {
			toolExe = tool + ".exe"
		} else {
			toolExe = tool
		}

		entries := osArchEntries[osArch]
		for _, e := range entries {
			u := baseURL + "/" + url.PathEscape(e.Filename)

			var digest []byte
			var exePath string
			if digest, exePath, err = processGenerateAsset(w, u, toolExe, hsh, checksums); err != nil {
				return err
			}

			if exePath != "" {
				exePaths[osArch] = exePath
			}
			fmt.Printf("--url %s=%s#sha256-%x\n", osArch, u, digest)
		}
	}

	for _, ep := range generateExePathArgs(exePaths) {
		fmt.Printf("--archive-exe-path %s\n", ep)
	}

	return nil
}
