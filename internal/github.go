package util

import (
	"regexp"
)

type GitHubRelease struct {
	TagName string               `json:"tag_name"`
	Assets  []GitHubReleaseAsset `json:"assets"`
}

type GitHubReleaseAsset struct {
	BrowserDownloadURL string `json:"browser_download_url"`
}

const osPart = `-(aix|darwin|dragonfly|(?:free|net|open)bsd|illumos|linux|plan9|solaris|windows)`
const extPart = `\.(?:exe|tar\.[gx]z|zip)?$`

var osArchArchiveRE = regexp.MustCompile(
	osPart + `[_-](amd64|arm(?:v7|64)?|mips(?:64)?(?:le)?|ppc64(?:le)?|riscv64|s390x|x86_64)` + extPart)
var osArchArchiveSecondaryRE = regexp.MustCompile(
	osPart + `[_-](armv6)` + extPart)

func (r GitHubRelease) PreferredOsArchReleaseAssets() (map[string]GitHubReleaseAsset, []GitHubReleaseAsset) {
	hits := make(map[string]GitHubReleaseAsset, len(r.Assets))
	pass2 := make([]GitHubReleaseAsset, 0, len(r.Assets)/3)
	for _, asset := range r.Assets {
		if m := osArchArchiveRE.FindStringSubmatch(asset.BrowserDownloadURL); m != nil {
			arch := m[2]
			switch arch {
			case "armv7":
				arch = "arm"
			case "x86_64":
				arch = "amd64"
			}
			hits[m[1]+"/"+arch] = asset
		} else {
			pass2 = append(pass2, asset)
		}
	}
	misses := make([]GitHubReleaseAsset, 0, len(pass2))
	for _, asset := range pass2 {
		if m := osArchArchiveSecondaryRE.FindStringSubmatch(asset.BrowserDownloadURL); m != nil {
			arch := m[2]
			switch arch {
			case "armv6":
				arch = "arm"
			}
			osArch := m[1] + "/" + arch
			if _, found := hits[osArch]; !found {
				hits[osArch] = asset
			}
		} else {
			misses = append(misses, asset)
		}
	}
	return hits, misses
}
