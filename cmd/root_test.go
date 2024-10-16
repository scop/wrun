// Copyright 2023 Ville Skyttä
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
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scop/wrun/internal/hashes"
)

func mustParseURL(t *testing.T, s string) *url.URL {
	t.Helper()
	ur, err := url.Parse(s)
	require.NoError(t, err)

	return ur
}

func Test_parseFlags(t *testing.T) {
	url1 := "https://example.com/os1-arch1"
	url2 := "https://example.com/os2-arch2"
	urld := "https://example.com/default-no-pattern"
	urlArgs := []string{
		"os1/arch1=" + url1,
		"os2/arch2=" + url2,
		urld,
	}
	exePathArgs := []string{}
	want := &rootCmdConfig{
		urlMatches: []urlMatch{
			{
				pattern: "os1/arch1",
				url:     mustParseURL(t, url1),
			},
			{
				pattern: "os2/arch2",
				url:     mustParseURL(t, url2),
			},
			{
				pattern: "*/*",
				url:     mustParseURL(t, urld),
			},
		},
		archiveExePathMatches: nil,
	}
	cfg := &rootCmdConfig{}
	err := parseFlags(cfg, urlArgs, exePathArgs)
	require.NoError(t, err)
	assert.Equal(t, want, cfg)
}

func Test_selectURL(t *testing.T) {
	const base = "https://example.com/"
	urlMatches := []urlMatch{
		{
			pattern: "linux/amd64",
			url:     mustParseURL(t, base+"linux-amd64"),
		},
		{
			pattern: "linux/arm64",
			url:     mustParseURL(t, base+"linux-arm64"),
		},
		{
			pattern: "darwin/arm64",
			url:     mustParseURL(t, base+"darwin-arm64"),
		},
		{
			pattern: "darwin/*",
			url:     mustParseURL(t, base+"darwin"),
		},
		{
			pattern: "linux/*",
			url:     mustParseURL(t, base+"linux"),
		},
		{
			pattern: "*/386",
			url:     mustParseURL(t, base+"386"),
		},
		{
			pattern: "*/*",
			url:     mustParseURL(t, base+"generic"),
		},
	}

	tests := []struct {
		osArch string
		want   string
	}{
		{
			osArch: "linux/amd64",
			want:   base + "linux-amd64",
		},
		{
			osArch: "linux/unknown",
			want:   base + "linux",
		},
		{
			osArch: "windows/386",
			want:   base + "386",
		},
		{
			osArch: "darwin/amd64",
			want:   base + "darwin",
		},
		{
			osArch: "windows/unknown",
			want:   base + "generic",
		},
		{
			osArch: "unknown/unknown",
			want:   base + "generic",
		},
	}
	for _, tt := range tests {
		t.Run(tt.osArch, func(t *testing.T) {
			ur, err := selectURL(tt.osArch, urlMatches)
			require.NoError(t, err)
			assert.Equal(t, tt.want, ur.String())
		})
	}
}

func Test_selectArchiveExePath(t *testing.T) {
	matches := []archiveExePathMatch{
		{
			pattern: "linux/amd64",
			exePath: "path/to/exe-linux-amd64",
		},
		{
			pattern: "linux/arm64",
			exePath: "path/to/exe-linux-arm64",
		},
		{
			pattern: "darwin/arm64",
			exePath: "path/to/exe-darwin-arm64",
		},
		{
			pattern: "darwin/*",
			exePath: "path/to/exe-darwin",
		},
		{
			pattern: "*/386",
			exePath: "path/to/exe-386",
		},
		{
			pattern: "*/amd64",
			exePath: "path/to/exe-amd64.exe",
		},
		{
			pattern: "*/*",
			exePath: "path/to/exe-generic",
		},
	}

	tests := []struct {
		osArch string
		want   string
	}{
		{
			osArch: "linux/amd64",
			want:   "path/to/exe-linux-amd64",
		},
		{
			osArch: "linux/unknown",
			want:   "path/to/exe-generic",
		},
		{
			osArch: "linux/386",
			want:   "path/to/exe-386",
		},
		{
			osArch: "windows/386",
			want:   "path/to/exe-386.exe",
		},
		{
			osArch: "windows/amd64",
			want:   "path/to/exe-amd64.exe",
		},
		{
			osArch: "darwin/amd64",
			want:   "path/to/exe-darwin",
		},
		{
			osArch: "darwin/arm64",
			want:   "path/to/exe-darwin-arm64",
		},
		{
			osArch: "aix/ppc64",
			want:   "path/to/exe-generic",
		},
	}
	for _, tt := range tests {
		t.Run(tt.osArch, func(t *testing.T) {
			p, err := selectArchiveExePath(tt.osArch, matches)
			require.NoError(t, err)
			assert.Equal(t, tt.want, p)
		})
	}
}

func Test_urlDir(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{
			"https://example.com/path/to/file",
			"example.com/path/to/file/" + cacheDirDigestPlaceholder,
		},
		{
			"http://example.com/path/to/file",
			"example.com/path/to/file/" + cacheDirDigestPlaceholder,
		},
		{
			"https://example.com//path/to/file/",
			"example.com/path/to/file/" + cacheDirDigestPlaceholder,
		},
		{
			"https://example.com/path/to/file?foo=bar",
			"example.com/path/to/file" + url.PathEscape("?foo=bar") + "/" + cacheDirDigestPlaceholder,
		},
		{
			"https://example.com//path/to/file#sha256-2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
			"example.com/path/to/file/sha256-2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
		},
		{
			"https://example.com/path/to/file?a=1&b=2#md5-7d793037a0760186574b0282f2f435e7",
			"example.com/path/to/file" + url.PathEscape("?a=1&b=2") + "/md5-7d793037a0760186574b0282f2f435e7",
		},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			u, err := url.Parse(tt.url)
			require.NoError(t, err)
			h, digest, err := hashes.ParseHashFragment(u.Fragment)
			require.NoError(t, err)
			assert.Equal(t, tt.want, urlDir(u, h, digest))
		})
	}
}

func Test_prepareHash(t *testing.T) {
	tests := []struct {
		fragment  string
		wantHash  crypto.Hash
		wantInErr string
	}{
		{
			"sha256-2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
			crypto.SHA256,
			"",
		},
		{
			"sha256-",
			0,
			"invalid fragment",
		},
		{
			"-2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
			0,
			"invalid fragment",
		},
		{
			"deadbeef",
			0,
			"invalid fragment",
		},
	}
	for _, tt := range tests {
		t.Run(tt.fragment, func(t *testing.T) {
			hash, _, err := hashes.ParseHashFragment(tt.fragment)
			if tt.wantInErr == "" {
				require.NoError(t, err)
				assert.Equal(t, tt.wantHash, hash)
			} else {
				require.ErrorContains(t, err, tt.wantInErr)
			}
		})
	}
}
