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
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/scop/wrun/internal/checksums"
	"github.com/scop/wrun/internal/github"
	"github.com/spf13/cobra"
)

func generateArbitraryGitHubProjectCommand(w *Wrun) *cobra.Command {
	var tool, release string
	genCmd := &cobra.Command{
		Use:   "github OWNER [PROJECT]",
		Short: "generate wrun command line arguments for tool in GitHub project asset",
		Example: strings.TrimSpace("" +
			w.ProgName + " generate github aquasecurity trivy\n" +
			w.ProgName + " generate github astral-sh ruff\n" +
			w.ProgName + " generate github daveshanley vacuum\n" +
			w.ProgName + " generate github dprint\n" +
			w.ProgName + " generate github golangci golangci-lint\n" +
			w.ProgName + " generate github hadolint\n" +
			w.ProgName + " generate github mvdan sh --tool shfmt\n" +
			w.ProgName + " generate github opentofu --tool tofu\n" +
			""),
		ValidArgsFunction: cobra.NoFileCompletions,
		Args:              cobra.RangeArgs(1, 2),
		Run: func(_ *cobra.Command, args []string) {
			if len(args) == 1 { // Default project = owner
				args = append(args, args[0])
			}
			if tool == "" {
				tool = args[1] // Default tool = project
			}
			if err := runGenerateGitHubProject(w, args[0], args[1], tool, release, nil); err != nil {
				w.LogError("%s", err)
				os.Exit(1)
			}
		},
	}
	genCmd.Flags().StringVarP(&tool, "tool", "T", "", "tool name to search within archive, defaults to project name")
	if err := genCmd.RegisterFlagCompletionFunc("tool", cobra.NoFileCompletions); err != nil {
		w.LogBug("register --tool completion: %v", err)
	}
	genCmd.Flags().StringVarP(&release, "release", "r", "", "project release version, defaults to automatically selected")
	if err := genCmd.RegisterFlagCompletionFunc("release", gitHubVersionCompleter(w, "", "")); err != nil {
		w.LogBug("register --release completion: %s", err)
	}

	return genCmd
}

func gitHubVersionCompleter(w *Wrun, owner, project string) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if owner == "" {
			owner = args[0]
		}
		if project == "" {
			project = owner
			if project == "" && len(args) == 1 {
				project = args[0]
			}
		}

		releases, err := releasesFromGitHubAPI(w, owner, project)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		ret := make([]string, 0, len(releases))
		for _, r := range releases {
			if strings.HasPrefix(r.TagName, toComplete) {
				ret = append(ret, r.TagName)
			}
		}
		return ret, cobra.ShellCompDirectiveNoFileComp
	}
}

func generateGitHubProjectCommand(w *Wrun, owner, project, tool string, osArchOverrideREs map[string]*regexp.Regexp) *cobra.Command {
	var release string
	genCmd := &cobra.Command{
		Use:               tool,
		Short:             "generate wrun command line arguments for " + tool,
		ValidArgsFunction: cobra.NoFileCompletions,
		Args:              cobra.NoArgs,
		Run: func(_ *cobra.Command, _ []string) {
			if err := runGenerateGitHubProject(w, owner, project, tool, release, osArchOverrideREs); err != nil {
				w.LogError("%s", err)
				os.Exit(1)
			}
		},
	}
	genCmd.Flags().StringVarP(&release, "release", "r", "", "project release version, defaults to automatically selected")
	if err := genCmd.RegisterFlagCompletionFunc("release", gitHubVersionCompleter(w, owner, project)); err != nil {
		w.LogBug("register --release completion: %s", err)
	}

	return genCmd
}

func releasesFromGitHubAPI(w *Wrun, owner, project string) ([]github.Release, error) {
	// Note: response is paginated, apparently 30 per page, and 100 is the max it can be bumped to.
	// Not really a problem for version autoselection, but may raise an eyebrow for release completions, even if not really a problem there either.
	const perPage = 100
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?per_page=%d", url.PathEscape(owner), url.PathEscape(project), perPage)
	resp, err := w.HTTPGet(url, "X-GitHub-Api-Version:2022-11-28", "Accept:application/vnd.github+json")
	if err != nil {
		return nil, err
	}
	var rels []github.Release
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

func preferredRelease(rels []github.Release) github.Release {
	// TODO: we may want to check that the release contains some usable assets, too; not all do

	// Prefer first non-draft non-prerelease, followed by the first non-draft prerelease
	var rel github.Release
	relFound := false
	for _, r := range rels {
		if r.Draft {
			continue
		}
		if !r.Prerelease {
			relFound = true
			rel = r
			break
		}
		if !relFound { // Keep earlier seen prerelease
			relFound = true
			rel = r
		}
	}
	if !relFound {
		// Fall back to first
		rel = rels[0]
	}
	return rel
}

func runGenerateGitHubProject(w *Wrun, owner, project, tool, version string, osArchOverrideREs map[string]*regexp.Regexp) error {
	var rel github.Release
	var err error
	if version == "" {
		rels, err := releasesFromGitHubAPI(w, owner, project)
		if err != nil {
			return fmt.Errorf("get %s/%s releases: %w", owner, project, err)
		}
		rel = preferredRelease(rels)
	} else {
		rel, err = releaseFromGitHubAPI(w, owner, project, version)
		if err != nil {
			return fmt.Errorf("get %s/%s release %s: %w", owner, project, version, err)
		}
	}

	osArchAssets, sumsAssets, unknownAssets := rel.PreferredOsArchReleaseAssets(osArchOverrideREs)
	for _, asset := range unknownAssets {
		w.LogInfo("no matching pattern for %q, ignoring", asset.BrowserDownloadURL)
	}
	var csums checksums.Checksums
	var buf bytes.Buffer
	for _, asset := range sumsAssets {
		if resp, err := w.HTTPGet(asset.BrowserDownloadURL); err != nil {
			return err
		} else if err := w.Download(resp, &buf, nil, nil); err != nil {
			return fmt.Errorf("download: %w", err)
		}
		if err = csums.UnmarshalText(buf.Bytes()); err != nil {
			w.LogWarn("unmarshal checksums from %q: %v", asset.BrowserDownloadURL, err)
		}
		buf.Reset()
	}

	// Process os/arch assets sorted by os/arch for stable output between runs
	osArchs := make([]string, 0, len(osArchAssets))
	for osArch := range osArchAssets {
		osArchs = append(osArchs, osArch)
	}
	slices.Sort(osArchs)

	exePaths := make(map[string]string, len(osArchs))

	hsh := crypto.SHA256.New()
	for _, osArch := range osArchs {
		asset := osArchAssets[osArch]
		if asset.State != github.ReleaseAssetStateUploaded {
			// TODO refresh state from API? What does GH give if one tries to download an "open" asset?
			return fmt.Errorf("asset with download URL %q state %q, expected %q", asset.BrowserDownloadURL, asset.State, github.ReleaseAssetStateUploaded)
		}

		var digest []byte
		var exePath string
		var toolExe string
		if strings.HasPrefix(osArch, "windows/") {
			toolExe = tool + ".exe"
		} else {
			toolExe = tool
		}
		if digest, exePath, err = processGenerateAsset(w, asset.BrowserDownloadURL, toolExe, hsh, csums); err != nil {
			return err
		}

		if exePath != "" {
			exePaths[osArch] = exePath
		}
		fmt.Printf("--url %s=%s#sha256-%x\n", osArch, asset.BrowserDownloadURL, digest)
	}

	for _, ep := range generateExePathArgs(exePaths) {
		fmt.Printf("--archive-exe-path %s\n", ep)
	}

	return nil
}
