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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/antchfx/xpath"
	util "github.com/scop/wrun/internal"
	"github.com/scop/wrun/internal/github"
	"github.com/scop/wrun/internal/pypi"
	"github.com/spf13/cobra"
)

func generateCommand(w *Wrun) *cobra.Command {
	genCmd := &cobra.Command{
		Use:   "generate",
		Short: "generate wrun command line arguments for various tools",
		Args:  cobra.NoArgs,
	}
	// TODO expose generic GH, PyPI generator commands
	genCmd.AddCommand(
		generateBlackCommand(w),
		generateCommittedCommand(w),
		generateRuffCommand(w),
		generateTyposCommand(w),
		generateVacuumCommand(w),
	)
	return genCmd
}

func versionsFromAtom(w *Wrun, url string) ([]string, error) {
	resp, err := w.HTTPGet(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	doc, err := xmlquery.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse atom from %s: %w", url, err)
	}
	nsMap := map[string]string{
		"atom": "http://www.w3.org/2005/Atom",
	}
	// For GitHub releases, title might differ from tag, look up from id
	expr, err := xpath.CompileWithNS("//atom:entry/atom:id", nsMap)
	if err != nil {
		return nil, fmt.Errorf("compile xpath: %w", err)
	}
	nodes := xmlquery.QuerySelectorAll(doc, expr)
	nn := len(nodes)
	if nn == 0 {
		return nil, fmt.Errorf("no versions found from %s", url)
	}

	versions := make([]string, 0, nn)
	for _, n := range nodes {
		// Sloppy, expected e.g. tag:github.com,2008:Repository/6731432/v0.10.0
		version = n.InnerText()
		if ix := strings.LastIndex(version, "/"); ix != -1 {
			version = version[ix+1:]
		}
		versions = append(versions, version)
	}

	return versions, nil
}

func versionsFromRSS2(w *Wrun, url string) ([]string, error) {
	resp, err := w.HTTPGet(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	doc, err := xmlquery.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse RSS 2 from %s: %w", url, err)
	}
	nodes, err := xmlquery.QueryAll(doc, "//channel/item/title")
	if err != nil {
		return nil, fmt.Errorf("query xpath: %w", err)
	}
	nn := len(nodes)
	if nn == 0 {
		return nil, fmt.Errorf("no versions found from %s", url)
	}

	versions := make([]string, 0, nn)
	for _, n := range nodes {
		versions = append(versions, n.InnerText())
	}

	return versions, nil
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

func generatePyPIProjectCommand(w *Wrun, projectName string) *cobra.Command {
	genCmd := &cobra.Command{
		Use:   projectName + " [VERSION]",
		Short: "generate wrun command line arguments for " + projectName,
		Args:  cobra.NoArgs,
		Run: func(_ *cobra.Command, args []string) {
			if err := runGeneratePyPIProject(w, projectName, args); err != nil {
				w.LogError("%s", err)
				os.Exit(1)
			}
		},
	}

	return genCmd
}

func runGeneratePyPIProject(w *Wrun, projectName string, args []string) error {
	var version string
	if len(args) != 0 {
		version = args[0]
	} else {
		vs, err := versionsFromRSS2(w, fmt.Sprintf("https://pypi.org/rss/project/%s/releases.xml", url.PathEscape(projectName)))
		if err != nil {
			return fmt.Errorf("get %s versions: %s", projectName, err)
		}
		version = vs[0]
	}
	version = strings.TrimPrefix(version, "v")

	url := fmt.Sprintf("https://pypi.org/pypi/%s/%s/json", url.PathEscape(projectName), url.PathEscape(version))
	resp, err := w.HTTPGet(url)
	if err != nil {
		return fmt.Errorf("get %s release info: %w", projectName, err)
	}

	var rel pypi.Release
	err = json.NewDecoder(resp.Body).Decode(&rel)
	if cErr := resp.Body.Close(); cErr != nil {
		w.LogWarn("close %s body: %v", url, cErr)
	}
	if err != nil {
		return fmt.Errorf("decode %s release info: %w", projectName, err)
	}

	osArchURLs, urlMisses := rel.PreferredOsArchReleaseURLs()
	for _, url := range urlMisses {
		w.LogWarn("no matching pattern for %q, ignoring", url.Filename)
	}
	exePaths := make([]string, 0, len(osArchURLs))

	// Process os/arch assets sorted by os/arch for stable output
	osArchs := make([]string, 0, len(osArchURLs))
	for osArch := range osArchURLs {
		osArchs = append(osArchs, osArch)
	}
	slices.Sort(osArchs)

	hshType := crypto.SHA256
	hsh := hshType.New()
	for _, osArch := range osArchs {
		pu := osArchURLs[osArch]
		if pu.URL == "" {
			w.LogWarn("missing URL for %q, ignoring", pu.Filename)
			continue
		}
		if pu.Digests.SHA256 == "" {
			w.LogWarn("missing SHA-256 digest for %q, ignoring", pu.Filename)
			continue
		}
		expectedDigest, err := hex.DecodeString(pu.Digests.SHA256)
		if err != nil {
			return fmt.Errorf("decode hex digest: %w", err)
		}

		// TODO skip download if skip-verify given
		if resp, err = w.HTTPGet(pu.URL); err != nil {
			return err
		}
		if err = w.Download(resp, nil, hsh, expectedDigest); err != nil {
			return fmt.Errorf("download: %w", err)
		}
		hsh.Reset()

		fmt.Printf("--url %s=%s#sha256-%s\n", osArch, pu.URL, pu.Digests.SHA256)
		ext := ""
		if strings.HasPrefix(osArch, "windows/") {
			ext = ".exe"
		}
		exePaths = append(exePaths, fmt.Sprintf("--archive-exe-path %s=%s-%s.data/scripts/%s%s", osArch, projectName, version, projectName, ext))
	}
	for _, ep := range exePaths {
		fmt.Println(ep)
	}

	return nil
}

func generateGitHubProjectCommand(w *Wrun, owner, project string) *cobra.Command {
	genCmd := &cobra.Command{
		Use:   project + " [VERSION]",
		Short: "generate wrun command line arguments for " + project,
		Args:  cobra.NoArgs,
		Run: func(_ *cobra.Command, args []string) {
			if err := runGenerateGitHubProject(w, owner, project, args); err != nil {
				w.LogError("%s", err)
				os.Exit(1)
			}
		},
	}

	return genCmd
}

func runGenerateGitHubProject(w *Wrun, owner, project string, args []string) error {
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
		rel = rels[0]
	}

	osArchAssets, assetMisses, sumsAssets := rel.PreferredOsArchReleaseAssets()
	for _, asset := range assetMisses {
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

	hshType := crypto.SHA256
	hsh := hshType.New()
	for _, osArch := range osArchs {
		asset := osArchAssets[osArch]
		if asset.State != github.ReleaseAssetStateUploaded {
			// TODO refresh state from API? What does GH give if one tries to download an "open" asset?
			return fmt.Errorf("asset with download URL %q state %q, expected %q", asset.BrowserDownloadURL, asset.State, github.ReleaseAssetStateUploaded)
		}
		if resp, err := w.HTTPGet(asset.BrowserDownloadURL); err != nil {
			return err
		} else if err = w.Download(resp, nil, hsh, nil); err != nil {
			return fmt.Errorf("download: %w", err)
		}
		digest := hsh.Sum(nil)
		hsh.Reset()

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
				w.LogWarn("no upstream checksum for %q", asset.BrowserDownloadURL)
			} else if !matchFound {
				return fmt.Errorf("")
			}
		}

		fmt.Printf("--url %s=%s#sha256-%x\n", osArch, asset.BrowserDownloadURL, digest)
	}

	return nil
}
