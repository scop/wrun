package github

import (
	"github.com/scop/wrun/internal/files"
)

type Release struct {
	TagName    string         `json:"tag_name"`
	Draft      bool           `json:"draft"`
	Prerelease bool           `json:"prerelease"`
	Assets     []ReleaseAsset `json:"assets"`
}

type ReleaseAssetState = string

const (
	ReleaseAssetStateOpen     ReleaseAssetState = "open"
	ReleaseAssetStateUploaded ReleaseAssetState = "uploaded"
)

type ReleaseAsset struct {
	BrowserDownloadURL string            `json:"browser_download_url"`
	State              ReleaseAssetState `json:"state"`

	// There is "name" available in the schema, but it's somewhat doubtful if that would be a better match for our os/arch mapping heuristics and checksum verification purposes than BrowserDownloadURL.
	// Docs on "name" are scarce, but the example "Team environment" https://docs.github.com/en/rest/releases/assets?apiVersion=2022-11-28#get-a-release-asset gives as of 2024-07-13 hints that it might not be.
	// Hence ignore it altogether here, so we are forced to be consistent and use BrowserDownloadURL for both above mentioned purposes.
}

func (r Release) PreferredOsArchReleaseAssets() (osArchAssets map[string]ReleaseAsset, checksumAssets []ReleaseAsset, unknownAssets []ReleaseAsset) {
	urlAssets := make(map[string]ReleaseAsset, len(r.Assets))
	for _, asset := range r.Assets {
		urlAssets[asset.BrowserDownloadURL] = asset
	}

	osArchAssets, checksumAssets, unknownAssets = files.Categorize(urlAssets)
	return
}
