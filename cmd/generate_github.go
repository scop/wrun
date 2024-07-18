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
	"crypto"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/klauspost/compress/zip"
	"github.com/mholt/archiver/v3"
	util "github.com/scop/wrun/internal"
	"github.com/scop/wrun/internal/files"
	"github.com/scop/wrun/internal/github"
	"github.com/spf13/cobra"
)

func generateArbitraryGitHubProjectCommand(w *Wrun) *cobra.Command {
	genCmd := &cobra.Command{
		Use:   "github OWNER PROJECT TOOL [VERSION]",
		Short: "generate wrun command line arguments for a GitHub project",
		Args:  cobra.RangeArgs(3, 4),
		Run: func(_ *cobra.Command, args []string) {
			if err := runGenerateGitHubProject(w, args[0], args[1], args[2], args[3:]); err != nil {
				w.LogError("%s", err)
				os.Exit(1)
			}
		},
	}

	return genCmd
}

func releasesFromGitHubAPI(w *Wrun, owner, project string) ([]github.Release, error) {
	const n = 10
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?per_page=%d", url.PathEscape(owner), url.PathEscape(project), n)
	resp, err := w.HTTPGet(url, "X-GitHub-Api-Version:2022-11-28", "Accept:application/vnd.github+json")
	if err != nil {
		return nil, err
	}
	rels := make([]github.Release, 0, n)
	err = json.NewDecoder(resp.Body).Decode(&rels)
	if cErr := resp.Body.Close(); cErr != nil {
		w.LogWarn("close %s body: %v", url, cErr)
	}
	if err != nil {
		return nil, fmt.Errorf("decode %s release info: %w", url, err)
	}
	return rels, nil
}

func releaseFromGitHubAPI(w *Wrun, owner, project, version string) (github.Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", url.PathEscape(owner), url.PathEscape(project), url.PathEscape(version))
	resp, err := w.HTTPGet(url, "X-GitHub-Api-Version:2022-11-28", "Accept:application/vnd.github+json")
	if err != nil {
		return github.Release{}, err
	}
	var rel github.Release
	err = json.NewDecoder(resp.Body).Decode(&rel)
	if cErr := resp.Body.Close(); cErr != nil {
		w.LogWarn("close %s body: %v", url, cErr)
	}
	if err != nil {
		return github.Release{}, fmt.Errorf("decode %s release info: %w", url, err)
	}
	return rel, nil
}

func generateGitHubProjectCommand(w *Wrun, owner, project, tool string) *cobra.Command {
	genCmd := &cobra.Command{
		Use:   tool + " [VERSION]",
		Short: "generate wrun command line arguments for " + tool,
		Args:  cobra.MaximumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			if err := runGenerateGitHubProject(w, owner, project, tool, args); err != nil {
				w.LogError("%s", err)
				os.Exit(1)
			}
		},
	}

	return genCmd
}

func runGenerateGitHubProject(w *Wrun, owner, project, tool string, args []string) error {
	var rel github.Release
	var err error
	if len(args) != 0 {
		version = args[0]
		rel, err = releaseFromGitHubAPI(w, owner, project, version)
		if err != nil {
			return fmt.Errorf("get %s/%s release %s: %w", owner, project, version, err)
		}
	} else {
		rels, err := releasesFromGitHubAPI(w, owner, project)
		if err != nil {
			return fmt.Errorf("get %s/%s releases: %w", owner, project, err)
		}
		relFound := false
		// TODO: loop through releases until we find one having some os/arch mapped assets. E.g. hadolint used to have latest release with none
		for _, rel = range rels {
			if !rel.Draft && !rel.Prerelease {
				relFound = true
				break
			}
		}
		if !relFound {
			rel = rels[0]
		}
	}

	osArchAssets, sumsAssets, unknownAssets := rel.PreferredOsArchReleaseAssets()
	for _, asset := range unknownAssets {
		w.LogWarn("no matching pattern for %q, ignoring", asset.BrowserDownloadURL)
	}
	var checksums util.Checksums
	var buf bytes.Buffer
	for _, asset := range sumsAssets {
		if resp, err := w.HTTPGet(asset.BrowserDownloadURL); err != nil {
			return err
		} else if err := w.Download(resp, &buf, nil, nil); err != nil {
			return fmt.Errorf("download: %w", err)
		}
		if err = checksums.UnmarshalText(buf.Bytes()); err != nil {
			w.LogWarn("unmarshal checksums from %q: %v", asset.BrowserDownloadURL, err)
		}
		buf.Reset()
	}
	haveChecksums := len(checksums.Entries) != 0

	// Process os/arch assets sorted by os/arch for stable output
	osArchs := make([]string, 0, len(osArchAssets))
	for osArch := range osArchAssets {
		osArchs = append(osArchs, osArch)
	}
	slices.Sort(osArchs)

	exePaths := make(map[string]string, len(osArchs))

	hshType := crypto.SHA256
	hsh := hshType.New()
	for _, osArch := range osArchs {
		asset := osArchAssets[osArch]
		if asset.State != github.ReleaseAssetStateUploaded {
			// TODO refresh state from API? What does GH give if one tries to download an "open" asset?
			return fmt.Errorf("asset with download URL %q state %q, expected %q", asset.BrowserDownloadURL, asset.State, github.ReleaseAssetStateUploaded)
		}

		resp, err := w.HTTPGet(asset.BrowserDownloadURL)
		if err != nil {
			return err
		}

		// Use temp filename _prefix_, archiver recognizes by filename extension
		tmpName := strings.ToLower(path.Base(asset.BrowserDownloadURL))
		if strings.HasSuffix(tmpName, ".whl") {
			tmpName += ".zip" // Make archiver recognize it
		}
		tmpf, err := os.CreateTemp("", "wrun*-"+tmpName)
		if err != nil {
			return fmt.Errorf("set up tempfile: %w", err) // TODO think this through
		}
		cleanUpTempFile := func() {
			if closeErr := tmpf.Close(); closeErr != nil && !errors.Is(closeErr, os.ErrClosed) {
				w.LogWarn("close tempfile: %v", closeErr)
			}
			if rmErr := os.Remove(tmpf.Name()); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) {
				w.LogWarn("remove tempfile: %v", rmErr)
			}
		}

		if err = w.Download(resp, tmpf, hsh, nil); err != nil {
			cleanUpTempFile()
			return fmt.Errorf("download: %w", err)
		}
		digest := hsh.Sum(nil)
		hsh.Reset()

		// TODO maybe if there's just one file in the archive, use it despite of tool name match?
		err = archiver.Walk(tmpf.Name(), func(f archiver.File) error {
			toolExe := tool
			if strings.HasPrefix(osArch, "windows/") {
				toolExe += ".exe"
			}
			if !f.IsDir() && f.Name() == toolExe {
				// Need to look in the archive type specific Header for the full path
				var path string
				switch fh := f.Header.(type) {
				case *tar.Header:
					path = fh.Name
				case zip.FileHeader:
					path = fh.Name
				default:
					// TODO think this through, should we emit an URL nevertheless, with a warnng etc?
					w.LogWarn("unsupported archive type for %q: %T", asset.BrowserDownloadURL, f.Header)
					return nil
				}
				// Prefer executables over others
				if files.HasExecutablePerms(f) {
					exePaths[osArch] = path
				} else if _, found := exePaths[osArch]; !found {
					exePaths[osArch] = path
				}
			}
			return nil
		})
		cleanUpTempFile()
		if err != nil {
			if !strings.Contains(err.Error(), "format unrecognized by filename") { // No better way as of archiver 3.5.1
				w.LogError("%v", err) // TODO think this through, continue or fail?
			}
		}

		// TODO refactor this for general reuse, e.g. in generate_terraform
		if haveChecksums {
			u, err := url.Parse(asset.BrowserDownloadURL)
			if err != nil {
				return fmt.Errorf("parse asset download URL %q for digest verification: %w", asset.BrowserDownloadURL, err)
			}
			fn := path.Base(u.Path)
			candidateFound := false
			matchFound := false
			if cs := checksums.Get(fn); cs != nil {
				for _, ce := range cs {
					if len(ce.Digest) == len(digest) {
						candidateFound = true
						if bytes.Equal(ce.Digest, digest) {
							w.LogInfo("digest match for %q: %x", asset.BrowserDownloadURL, ce.Digest)
							matchFound = true
							break
						} else {
							w.LogInfo("digest candidate for %q mismatch: expected %x, have %x", asset.BrowserDownloadURL, ce.Digest, digest)
						}
					} else {
						w.LogInfo("digest candidate for %q skipped due to length mismatch: %x, have %x", asset.BrowserDownloadURL, ce.Digest, digest)
					}
				}
			}
			if !candidateFound {
				w.LogWarn("no upstream digest for %q", asset.BrowserDownloadURL)
			} else if !matchFound {
				return fmt.Errorf("no digest match for %q", asset.BrowserDownloadURL)
			}
		}

		fmt.Printf("--url %s=%s#sha256-%x\n", osArch, asset.BrowserDownloadURL, digest)
	}

	if len(exePaths) != 0 {
		// Simplify output if path is the same in all archives (and if the tool was found in all of them).
		// If a tool was not found in some of the archives, warn about it and ignore but proceed. Output the url arg though in the warning so it can be fixed up manually by the user.
		var prevExePath string
		sameExePath := true
		for osArch, exePath := range exePaths {
			if strings.HasPrefix(osArch, "windows/") {
				// TODO trim case insensitive .exe
				exePath = strings.TrimSuffix(exePath, ".exe")
			}
			if prevExePath != "" {
				if prevExePath != exePath {
					sameExePath = false
					break
				}
			}
			prevExePath = exePath
		}
		if sameExePath {
			fmt.Printf("--archive-exe-path %s\n", prevExePath)
		} else {
			for _, osArch := range osArchs {
				if exePath, found := exePaths[osArch]; found {
					fmt.Printf("--archive-exe-path %s=%s\n", osArch, exePath)
				} // TODO else?
			}
		}
	}

	return nil
}
