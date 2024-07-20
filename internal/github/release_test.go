package github_test

import (
	"testing"

	"github.com/scop/wrun/internal/github"
	"github.com/stretchr/testify/assert"
)

func TestPreferredOsArchReleaseAssets_Basic(t *testing.T) {
	release := github.Release{
		Assets: []github.ReleaseAsset{
			{BrowserDownloadURL: "https://github.com/scop/wrun/releases/v1.0.0/checksums.txt"},
			{BrowserDownloadURL: "https://github.com/scop/wrun/releases/v1.0.0/example-1.0.0-darwin-arm64.tar.gz"},
			{BrowserDownloadURL: "https://github.com/scop/wrun/releases/v1.0.0/example-1.0.0-linux-amd64.deb"},
			{BrowserDownloadURL: "https://github.com/scop/wrun/releases/v1.0.0/example-1.0.0-linux-amd64.tar.gz"},
			{BrowserDownloadURL: "https://github.com/scop/wrun/releases/v1.0.0/example-1.0.0-linux-armv6.tar.gz"}, // not expected in hits or misses
			{BrowserDownloadURL: "https://github.com/scop/wrun/releases/v1.0.0/example-1.0.0-linux-armv7.tar.gz"}, // chosen instead of the armv7 one
			{BrowserDownloadURL: "https://github.com/scop/wrun/releases/v1.0.0/example-1.0.0-windows-amd64.zip"},
		},
	}
	expectedHits := map[string]github.ReleaseAsset{
		"darwin/arm64":  {BrowserDownloadURL: "https://github.com/scop/wrun/releases/v1.0.0/example-1.0.0-darwin-arm64.tar.gz"},
		"linux/amd64":   {BrowserDownloadURL: "https://github.com/scop/wrun/releases/v1.0.0/example-1.0.0-linux-amd64.tar.gz"},
		"linux/arm":     {BrowserDownloadURL: "https://github.com/scop/wrun/releases/v1.0.0/example-1.0.0-linux-armv7.tar.gz"},
		"windows/amd64": {BrowserDownloadURL: "https://github.com/scop/wrun/releases/v1.0.0/example-1.0.0-windows-amd64.zip"},
	}
	expectedSums := []github.ReleaseAsset{
		{BrowserDownloadURL: "https://github.com/scop/wrun/releases/v1.0.0/checksums.txt"},
	}
	expectedUnknown := []github.ReleaseAsset{
		{BrowserDownloadURL: "https://github.com/scop/wrun/releases/v1.0.0/example-1.0.0-linux-amd64.deb"},
	}

	// TODO some override tests

	hits, sums, unknown := release.PreferredOsArchReleaseAssets(nil)
	assert.Equal(t, expectedHits, hits)
	assert.ElementsMatch(t, expectedSums, sums)
	assert.ElementsMatch(t, expectedUnknown, unknown)
}

func TestPreferredOsArchReleaseAssets_NonArchive(t *testing.T) {
	release := github.Release{
		Assets: []github.ReleaseAsset{
			{BrowserDownloadURL: "https://github.com/mvdan/sh/releases/download/v3.8.0/sha256sums.txt"},
			{BrowserDownloadURL: "https://github.com/mvdan/sh/releases/download/v3.8.0/shfmt_v3.8.0_darwin_amd64"},
			{BrowserDownloadURL: "https://github.com/mvdan/sh/releases/download/v3.8.0/shfmt_v3.8.0_darwin_arm64"},
			{BrowserDownloadURL: "https://github.com/mvdan/sh/releases/download/v3.8.0/shfmt_v3.8.0_linux_386"},
			{BrowserDownloadURL: "https://github.com/mvdan/sh/releases/download/v3.8.0/shfmt_v3.8.0_linux_amd64"},
			{BrowserDownloadURL: "https://github.com/mvdan/sh/releases/download/v3.8.0/shfmt_v3.8.0_linux_arm"},
			{BrowserDownloadURL: "https://github.com/mvdan/sh/releases/download/v3.8.0/shfmt_v3.8.0_linux_arm64"},
			{BrowserDownloadURL: "https://github.com/mvdan/sh/releases/download/v3.8.0/shfmt_v3.8.0_windows_386.exe"},
			{BrowserDownloadURL: "https://github.com/mvdan/sh/releases/download/v3.8.0/shfmt_v3.8.0_windows_amd64.exe"},
		},
	}
	expectedHits := map[string]github.ReleaseAsset{
		"darwin/amd64":  {BrowserDownloadURL: "https://github.com/mvdan/sh/releases/download/v3.8.0/shfmt_v3.8.0_darwin_amd64"},
		"darwin/arm64":  {BrowserDownloadURL: "https://github.com/mvdan/sh/releases/download/v3.8.0/shfmt_v3.8.0_darwin_arm64"},
		"linux/386":     {BrowserDownloadURL: "https://github.com/mvdan/sh/releases/download/v3.8.0/shfmt_v3.8.0_linux_386"},
		"linux/amd64":   {BrowserDownloadURL: "https://github.com/mvdan/sh/releases/download/v3.8.0/shfmt_v3.8.0_linux_amd64"},
		"linux/arm":     {BrowserDownloadURL: "https://github.com/mvdan/sh/releases/download/v3.8.0/shfmt_v3.8.0_linux_arm"},
		"linux/arm64":   {BrowserDownloadURL: "https://github.com/mvdan/sh/releases/download/v3.8.0/shfmt_v3.8.0_linux_arm64"},
		"windows/386":   {BrowserDownloadURL: "https://github.com/mvdan/sh/releases/download/v3.8.0/shfmt_v3.8.0_windows_386.exe"},
		"windows/amd64": {BrowserDownloadURL: "https://github.com/mvdan/sh/releases/download/v3.8.0/shfmt_v3.8.0_windows_amd64.exe"},
	}
	expectedSums := []github.ReleaseAsset{
		{BrowserDownloadURL: "https://github.com/mvdan/sh/releases/download/v3.8.0/sha256sums.txt"},
	}
	expectedUnknown := []github.ReleaseAsset{}

	// TODO some override tests

	hits, sums, unknown := release.PreferredOsArchReleaseAssets(nil)
	assert.Equal(t, expectedHits, hits)
	assert.ElementsMatch(t, expectedSums, sums)
	assert.ElementsMatch(t, expectedUnknown, unknown)
}
