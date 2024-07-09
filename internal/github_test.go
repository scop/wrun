package util_test

import (
	"testing"

	util "github.com/scop/wrun/internal"
	"github.com/stretchr/testify/assert"
)

func TestPreferredOsArchReleaseAssets(t *testing.T) {
	// TODO test non-archive suffixes "", "exe", e.g. shfmt
	release := util.GitHubRelease{
		Assets: []util.GitHubReleaseAsset{
			{BrowserDownloadURL: "https://github.com/scop/wrun/releases/v1.0.0/example-v1.0.0-linux-amd64.tar.gz"},
			{BrowserDownloadURL: "https://github.com/scop/wrun/releases/v1.0.0/example-v1.0.0-linux-amd64.deb"},
			{BrowserDownloadURL: "https://github.com/scop/wrun/releases/v1.0.0/example-v1.0.0-darwin-arm64.tar.gz"},
			{BrowserDownloadURL: "https://github.com/scop/wrun/releases/v1.0.0/example-v1.0.0-linux-armv6.tar.gz"}, // not expected in hits or misses
			{BrowserDownloadURL: "https://github.com/scop/wrun/releases/v1.0.0/example-v1.0.0-linux-armv7.tar.gz"}, // chosen instead of the armv7 one
			{BrowserDownloadURL: "https://github.com/scop/wrun/releases/v1.0.0/checksums.txt"},
		},
	}
	expectedHits := map[string]util.GitHubReleaseAsset{
		"linux/amd64":  {BrowserDownloadURL: "https://github.com/scop/wrun/releases/v1.0.0/example-v1.0.0-linux-amd64.tar.gz"},
		"darwin/arm64": {BrowserDownloadURL: "https://github.com/scop/wrun/releases/v1.0.0/example-v1.0.0-darwin-arm64.tar.gz"},
		"linux/arm":    {BrowserDownloadURL: "https://github.com/scop/wrun/releases/v1.0.0/example-v1.0.0-linux-armv7.tar.gz"},
	}
	expectedMisses := []util.GitHubReleaseAsset{
		{BrowserDownloadURL: "https://github.com/scop/wrun/releases/v1.0.0/example-v1.0.0-linux-amd64.deb"},
		{BrowserDownloadURL: "https://github.com/scop/wrun/releases/v1.0.0/checksums.txt"},
	}

	hits, misses := release.PreferredOsArchReleaseAssets()
	assert.Equal(t, expectedHits, hits)
	assert.ElementsMatch(t, expectedMisses, misses)
}
