package pypi

import "path/filepath"

type Release struct {
	URLs []ReleaseURL `json:"urls"`
}

type ReleaseURL struct {
	PackageType string            `json:"packagetype"`
	Filename    string            `json:"filename"`
	URL         string            `json:"url"`
	Digests     ReleaseURLDigests `json:"digests"`
}

type ReleaseURLDigests struct {
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

func (r Release) PreferredOsArchReleaseURLs() (map[string]ReleaseURL, []ReleaseURL) {
	hits := make(map[string]ReleaseURL, len(r.URLs))
	misses := []ReleaseURL{}
urls:
	for _, url := range r.URLs {
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
