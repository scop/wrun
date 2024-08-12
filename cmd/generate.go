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
	"archive/tar"
	"bytes"
	"fmt"
	"hash"
	"net/url"
	"path"
	"slices"
	"strings"

	"github.com/klauspost/compress/zip"
	"github.com/mholt/archiver/v3"
	"github.com/scop/wrun/internal/checksums"
	"github.com/scop/wrun/internal/files"
	"github.com/spf13/cobra"
)

func generateCommand(w *Wrun) *cobra.Command {
	genCmd := &cobra.Command{
		Use:   "generate",
		Short: "generate wrun command line arguments for various tools",
		Args:  cobra.NoArgs,
	}
	genCmd.AddCommand(
		generateArbitraryGitHubProjectCommand(w),
		generateArbitraryPyPIProjectCommand(w),
		generateBlackCommand(w),
		generateShellcheckCommand(w),
		generateTerraformCommand(w),
	)
	return genCmd
}

func processGenerateAsset(w *Wrun, ur, tool string, hsh hash.Hash, csums checksums.Checksums) (digest []byte, exePath string, err error) {
	resp, err := w.HTTPGet(ur)
	if err != nil {
		return nil, "", err
	}

	tmpf, cleanUpTempFile, err := w.SetUpTempfile(ur, "")
	if err != nil {
		return nil, "", fmt.Errorf("set up tempfile: %w", err)
	}

	if err = w.Download(resp, tmpf, hsh, nil); err != nil {
		cleanUpTempFile()
		return nil, "", fmt.Errorf("download: %w", err)
	}
	digest = hsh.Sum(nil)
	hsh.Reset()

	exePath, err = findToolInArchive(tmpf.Name(), tool)
	cleanUpTempFile()
	if err != nil {
		if !strings.Contains(err.Error(), "format unrecognized by filename") { // No better way as of archiver 3.5.1
			w.LogError("find tool in archive: %v", err)
		}
	}

	if len(csums.Entries) != 0 {
		u, err := url.Parse(ur)
		if err != nil {
			return nil, "", fmt.Errorf("parse download URL %q for digest verification: %w", ur, err)
		}
		fn := path.Base(u.Path)
		candidateFound := false
		matchFound := false
		if cs := csums.Get(fn); cs != nil {
			for _, ce := range cs {
				if len(ce.Digest) == len(digest) {
					candidateFound = true
					if bytes.Equal(ce.Digest, digest) {
						w.LogInfo("digest match for %q: %x", ur, ce.Digest)
						matchFound = true
						break
					} else {
						w.LogInfo("digest candidate for %q mismatch: expected %x, have %x", ur, ce.Digest, digest)
					}
				} else {
					w.LogInfo("digest candidate for %q skipped due to length mismatch: %x, have %x", ur, ce.Digest, digest)
				}
			}
		}
		if !candidateFound {
			w.LogWarn("no upstream digest for %q", ur)
		} else if !matchFound {
			return nil, "", fmt.Errorf("no digest match for %q", ur)
		}
	}

	return digest, exePath, nil
}

func findToolInArchive(filename, toolExe string) (path string, err error) {
	// TODO maybe if there's just one file in the archive, use it despite of tool name match?
	err = archiver.Walk(filename, func(f archiver.File) error {
		if !f.IsDir() && f.Name() == toolExe {
			// Need to look in the archive type specific Header for the full path
			switch fh := f.Header.(type) {
			case *tar.Header:
				path = fh.Name
			case zip.FileHeader:
				path = fh.Name
			default:
				return fmt.Errorf("unsupported file header: %T", f.Header)
			}
			// Prefer executables over others
			if files.HasExecutablePerms(f) {
				return archiver.ErrStopWalk
			}
		}
		return nil
	})
	if err != nil {
		err = fmt.Errorf("walk archive: %w", err)
	}
	return
}

func generateExePathArgs(osArchExePaths map[string]string) []string {
	ret := make([]string, 0, len(osArchExePaths))
	if len(osArchExePaths) != 0 {
		// Simplify output if path is the same in all archives (and if the tool was found in all of them).
		// If a tool was not found in some of the archives, warn about it and ignore but proceed. Output the url arg though in the warning so it can be fixed up manually by the user.
		var prevExePath string
		sameExePath := true
		for osArch, exePath := range osArchExePaths {
			ret = append(ret, osArch+"="+exePath)
			if strings.HasPrefix(osArch, "windows/") && strings.HasSuffix(strings.ToLower(exePath), ".exe") {
				exePath = exePath[:len(exePath)-len(".exe")]
			}
			if prevExePath != "" && prevExePath != exePath {
				sameExePath = false
			}
			prevExePath = exePath
		}
		if sameExePath {
			ret = []string{prevExePath}
		}
	}
	slices.Sort(ret) // for stable output between runs
	return ret
}
