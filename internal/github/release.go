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

package github

import (
	"regexp"

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

func (r Release) PreferredOsArchReleaseAssets(osArchOverrideREs map[string]*regexp.Regexp) (osArchAssets map[string]ReleaseAsset, checksumAssets, otherAssets []ReleaseAsset) {
	urlAssets := make(map[string]ReleaseAsset, len(r.Assets))
	for _, asset := range r.Assets {
		urlAssets[asset.BrowserDownloadURL] = asset
	}

	osArchAssets, checksumAssets, otherAssets = files.Categorize(urlAssets, osArchOverrideREs)
	return
}
