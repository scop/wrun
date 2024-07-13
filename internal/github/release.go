package github

import (
	"regexp"
	"strings"
)

type Release struct {
	TagName string         `json:"tag_name"`
	Assets  []ReleaseAsset `json:"assets"`
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

// osPart is mostly generated with `generate_os_arch_regexps.py`
const osPart = `[-_](aix|android|darwin|dragonfly|freebsd|illumos|ios|js|linux|macos|netbsd|openbsd|plan9|solaris|wasip1|windows)`
const extPart = `(?:\.(?:exe|tar\.[gx]z|zip))?$`

var osArchArchiveREs = []*regexp.Regexp{
	regexp.MustCompile(
		// this list of archs is generated with `generate_os_arch_regexps.py`
		osPart + `[_-](386|amd64|arm|arm64|loong64|mips|mips64|mips64le|mipsle|ppc64|ppc64le|riscv64|s390x|wasm)` + extPart),
	regexp.MustCompile(
		osPart + `[_-](32bit|64bit|aarch64|armv7|i386|x86_64)` + extPart),
	regexp.MustCompile(
		osPart + `[_-](armv6|armv6hf)` + extPart),
}

var checksumsRE = regexp.MustCompile(`(?i)/[^/]*(?:(?:(?:check|sha256)sums)\.txt|[^/]\.sha256)$`)

func (r Release) PreferredOsArchReleaseAssets() (hits map[string]ReleaseAsset, misses []ReleaseAsset, checksums []ReleaseAsset) {
	hits = make(map[string]ReleaseAsset, len(r.Assets))
	work := make([]ReleaseAsset, len(r.Assets))
	if n := copy(work, r.Assets); n != len(r.Assets) {
		panic("unexpected number of assets copied")
	}

	for _, re := range osArchArchiveREs {
		misses = make([]ReleaseAsset, 0, len(work))
		for _, asset := range work {
			if m := re.FindStringSubmatch(strings.ToLower(asset.BrowserDownloadURL)); m != nil {
				os := m[1]
				switch os {
				case "macos":
					os = "darwin"
				}
				arch := m[2]
				switch arch {
				case "32bit", "i386":
					arch = "386"
				case "64bit", "x86_64":
					arch = "amd64"
				case "aarch64":
					arch = "arm64"
				case "armv6", "armv6hf", "armv7":
					arch = "arm"
				}
				osArch := os + "/" + arch
				if _, found := hits[osArch]; !found {
					hits[osArch] = asset
				}
			} else {
				misses = append(misses, asset)
			}
		}
		work = misses
	}

	misses = make([]ReleaseAsset, 0, len(work))
	for _, asset := range work {
		if checksumsRE.MatchString(asset.BrowserDownloadURL) {
			checksums = append(checksums, asset)
		} else {
			misses = append(misses, asset)
		}
	}

	return hits, misses, checksums
}
