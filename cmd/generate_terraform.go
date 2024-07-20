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
	osArchEntries, unknownEntries, _ := files.Categorize(fileEntries, nil)
	for _, ce := range unknownEntries {
		w.LogWarn("no matching pattern for %q, ignoring", ce)
	}

	hn, err := util.HashByName(util.HashName(crypto.SHA256))
	if err != nil {
		return err
	}
	hsh := hn.New()
	for osArch, ces := range osArchEntries {
		candidateFound := false
		matchFound := false
		// We should not have any empty slices, and the filename is the same for all "ces" entries
		u := baseURL + "/" + url.PathEscape(ces[0].Filename)
		for _, ce := range ces {
			if len(ce.Digest) != hsh.Size() {
				w.LogInfo("digest candidate for %q skipped due to length mismatch: %x, have %x", u, ce.Digest, hsh.Size())
			} else {
				candidateFound = true
				resp, err := w.HTTPGet(u)
				if err != nil {
					return err
				}
				if err = w.Download(resp, nil, hsh, nil); err != nil {
					return err
				}
				digest := hsh.Sum(nil)
				hsh.Reset()
				if bytes.Equal(digest, ce.Digest) {
					w.LogInfo("digest match for %q: %x", u, ce.Digest)
					matchFound = true
					fmt.Printf("--url %s=%s#sha256-%x\n", osArch, u, digest)
					break
				} else {
					w.LogInfo("digest candidate for %q mismatch: expected %x, have %x", u, ce.Digest, digest)
				}
			}
		}
		if !candidateFound {
			w.LogWarn("no upstream digest for %q", u)
		} else if !matchFound {
			return fmt.Errorf("no digest match for %q", u)
		}
	}

	// TODO archive exe stuff

	return nil
}
