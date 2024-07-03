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
	"path/filepath"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/antchfx/xpath"
	"github.com/spf13/cobra"
)

func generateCommand(w *Wrun) *cobra.Command {
	genCmd := &cobra.Command{
		Use:   "generate",
		Short: "generate wrun command line arguments for various tools",
		Args:  cobra.NoArgs,
	}
	genCmd.AddCommand(
		generateBlackCommand(w),
		generateCommittedCommand(w),
		generateRuffCommand(w),
		generateTyposCommand(w),
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

type PyPIRelease struct {
	URLs []PyPIReleaseURL `json:"urls"`
}

type PyPIReleaseURL struct {
	PackageType string                `json:"packagetype"`
	Filename    string                `json:"filename"`
	URL         string                `json:"url"`
	Digests     PyPIReleaseURLDigests `json:"digests"`
}

type PyPIReleaseURLDigests struct {
	SHA256 string `json:"sha256"`
}

/*
wrun: WARN: missing pattern for "ruff-0.5.0-py3-none-linux_armv6l.whl", ignoring
*/
var osArchWheels = map[string]string{
	"darwin/amd64":  "*-macosx_*_x86_64.whl",
	"darwin/arm64":  "*-macosx_*_arm64.whl",
	"linux/386":     "*-musllinux_*_i686.whl",
	"linux/amd64":   "*-musllinux_*_x86_64.whl",
	"linux/arm":     "*-musllinux_*_armv7l.whl",
	"linux/arm64":   "*-musllinux_*_aarch64.whl",
	"windows/386":   "*-win32.whl",
	"windows/amd64": "*-win_amd64.whl",
	"windows/arm64": "*-win_arm64.whl",
}

var osArchSecondaryWheels = map[string]string{
	"linux/386":     "*-manylinux_*_i686.manylinux*_i686.whl",
	"linux/amd64":   "*-manylinux_*_x86_64.manylinux*_x86_64.whl",
	"linux/arm":     "*-manylinux_*_armv7l.manylinux*_armv7l.whl",
	"linux/arm64":   "*-manylinux_*_aarch64.manylinux*_aarch64.whl",
	"linux/ppc64":   "*-manylinux_*_ppc64.manylinux*_ppc64.whl",
	"linux/ppc64le": "*-manylinux_*_ppc64le.manylinux*_ppc64le.whl",
	"linux/s390x":   "*-manylinux_*_s390x.manylinux*_s390x.whl",
}

func (p PyPIRelease) PreferredOsArchReleaseURLs() (map[string]PyPIReleaseURL, []PyPIReleaseURL) {
	hits := make(map[string]PyPIReleaseURL, len(p.URLs))
	misses := []PyPIReleaseURL{}
urls:
	for _, url := range p.URLs {
		if url.PackageType != "bdist_wheel" {
			continue
		}
		for osArch, pattern := range osArchWheels {
			if m, err := filepath.Match(pattern, url.Filename); err == nil && m {
				hits[osArch] = url
				continue urls
			}
		}
		for osArch, pattern := range osArchSecondaryWheels {
			// Try match first before existing osArch lookup for proper tracking of misses
			if m, err := filepath.Match(pattern, url.Filename); err == nil && m {
				if _, found := hits[osArch]; !found {
					hits[osArch] = url
				}
				continue urls
			}
		}
		misses = append(misses, url)
	}
	return hits, misses
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

	var rel PyPIRelease
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
	hshType := crypto.SHA256
	hsh := hshType.New()
	for osArch, pu := range osArchURLs {
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
		if err = w.Download(resp, nil, hshType, hsh, expectedDigest); err != nil {
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
