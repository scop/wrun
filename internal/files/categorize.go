package files

import (
	"regexp"
	"strings"
)

func Categorize[T any](fileAssets map[string]T) (osArchPreferred map[string]T, checksums []T, unknown []T) {

	// OS and arch parts slices are patterns to match in decreasing order of preference.
	// For example, we want to match musl linuxes before gnu ones for portability reasons, and similarly armv7 for arm before armv6 etc.

	// This code is expected to run at most once per wrun invocation, so generate regexps inline
	// instead of on init or such.

	osParts := []string{
		`(?P<os>aix|android|(?:apple-)?darwin|dragonfly|freebsd|illumos|ios|js|linux|macos|netbsd|openbsd|pc-windows-msvc|plan9|solaris|unknown-linux-musl|wasip1|windows)`,
		`(?P<os>unknown-linux-gnu)`,
	}
	archParts := []string{
		`(?P<arch>i?386|amd64|arm|arm64|loong64|mips|mips64|mips64le|mipsle|ppc64|ppc64le|riscv64|s390x|wasm|x86_64)`,
		`(?P<arch>32bit|64bit|aarch64|armv7)`,
		`(?P<arch>armv6|armv6hf)`,
	}
	const extPart = `(?:\.(?:exe|tar\.[gx]z|zip))?$`

	var osArchFileREs []*regexp.Regexp
	for _, osPart := range osParts {
		for _, archPart := range archParts {
			osArchFileREs = append(osArchFileREs,
				// We want lowercase matches for these, so don't do case insensitive but match against lowercased
				regexp.MustCompile("[_-]"+osPart+"[_-]"+archPart+extPart),
				regexp.MustCompile("[_-]"+archPart+"[_-]"+osPart+extPart),
			)
		}
	}

	checksumsRE := regexp.MustCompile(`(?i)/[^/]*(?:` +
		`sums[^/]*\.txt|` +
		`[^/]\.sha256` +
		`)$`)

	osArchPreferred = make(map[string]T, len(fileAssets))
	work := make([]string, 0, len(fileAssets))
	for k := range fileAssets {
		work = append(work, k)
	}

	for _, re := range osArchFileREs {
		unknownFiles := make([]string, 0, len(work))
		for _, name := range work {
			// ToLower so we get lowercase matches
			if m := re.FindStringSubmatch(strings.ToLower(name)); m != nil {
				var os, arch string
				for i, name := range re.SubexpNames() {
					switch name {
					case "":
						// whole expression
					case "os":
						os = m[i]
					case "arch":
						arch = m[i]
					default:
						panic("unhandled subexpression name: " + name)
					}
				}
				switch os {
				case "apple-darwin", "macos":
					os = "darwin"
				case "pc-windows-msvc":
					os = "windows"
				case "unknown-linux-gnu", "unknown-linux-musl":
					os = "linux"
				}
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
				if _, found := osArchPreferred[osArch]; !found {
					osArchPreferred[osArch] = fileAssets[name]
				}
			} else {
				unknownFiles = append(unknownFiles, name)
			}
		}
		work = unknownFiles
	}

	unknown = make([]T, 0, len(work))
	for _, name := range work {
		if checksumsRE.MatchString(name) {
			checksums = append(checksums, fileAssets[name])
		} else {
			unknown = append(unknown, fileAssets[name])
		}
	}

	return
}
