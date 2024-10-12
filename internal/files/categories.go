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

package files

import (
	"regexp"
	"strings"
)

// Categorize files assets to preferred ones for different operating systems and architectures, checksums, and other kinds.
//
// The fileAssets argument is a map with filenames as keys and assets as values.
//
// The overrides argument has OS/arch strings as keys,
// and regular expressions applied against filenames in fileAssets as values that are processed before the default categorization rules.
// If an override regular expression matches an asset filename, it will cause the asset to be treated as the preferred one for the OS/arch in the override.
//
// The returned osArchPreferred is has OS/arch strings as keys, and the corresponding assets as values.
// All given assets are present in at most one of the return values.
// The only ones that are not in any are ones that apply to an OS/architecture combination, but for which a more preferred one was found.
func Categorize[T any](fileAssets map[string]T, overrides map[string]*regexp.Regexp) (osArchPreferred map[string]T, checksums, others []T) {
	// OS and arch parts slices are patterns to match in decreasing order of preference.
	// For example, we want to match musl linuxes before gnu ones for portability reasons, and similarly armv7 for arm before armv6 etc.

	// This code is expected to run at most once per wrun invocation, so generate regexps inline
	// instead of on init or such.

	osParts := []string{
		`(?P<os>aix|android|(?:apple-)?darwin|dragonfly|freebsd|illumos|ios|js|linux|macos|netbsd|openbsd|pc-windows-msvc|plan9|solaris|unknown-linux-musl(?:eabihf)?|wasip1|windows)`,
		`(?P<os>unknown-linux-gnu(?:eabihf)?)`,
	}
	archParts := []string{
		`(?P<arch>i?[36]86|amd64|arm|arm64|loong64|mips|mips64|mips64le|mipsle|p(?:ower)?pc64(?:le)?|riscv64|s390x|wasm|x86_64)`,
		`(?P<arch>32bit|64bit|aarch64|armv7)`,
		`(?P<arch>armv6|armv6hf)`,
	}
	extParts := []string{ // Prefer tarballs over zips, mostly just for stable ordering, possibly also for smaller size at times
		`\.tar\.[gx]z$`,
		`(?:\.(?:exe|zip))?$`,
	}

	var osArchFileREs []*regexp.Regexp
	for _, osPart := range osParts {
		for _, archPart := range archParts {
			for _, extPart := range extParts {
				osArchFileREs = append(osArchFileREs,
					// We want lowercase matches for these, so don't do case insensitive but match against lowercased
					regexp.MustCompile("[_.-]"+osPart+"[_.-]"+archPart+extPart),
					regexp.MustCompile("[_.-]"+archPart+"[_.-]"+osPart+extPart),
				)
			}
		}
	}

	checksumsRE := regexp.MustCompile(`(?i)(?:^|/)[^/]*(?:` +
		`sums[^/]*\.txt|` +
		`[^/]\.(?:md5|sha(?:1|224|256|384|512))` +
		`)$`)

	osArchPreferred = make(map[string]T, len(fileAssets))
	work := make([]string, 0, len(fileAssets))
	for k := range fileAssets {
		work = append(work, k)
	}

	for osArch, re := range overrides {
		unknownFiles := make([]string, 0, len(work))
		for _, name := range work {
			if re.MatchString(name) {
				osArchPreferred[osArch] = fileAssets[name]
			} else {
				unknownFiles = append(unknownFiles, name)
			}
		}
		work = unknownFiles
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
						panic("wrun: BUG : unhandled subexpression name: " + name)
					}
				}
				switch os {
				case "apple-darwin", "macos":
					os = "darwin"
				case "pc-windows-msvc":
					os = "windows"
				case "unknown-linux-gnu", "unknown-linux-gnueabihf", "unknown-linux-musl", "unknown-linux-musleabihf":
					os = "linux"
				}
				switch arch {
				case "32bit", "i386", "686", "i686":
					arch = "386"
				case "64bit", "x86_64":
					arch = "amd64"
				case "aarch64":
					arch = "arm64"
				case "armv6", "armv6hf", "armv7":
					arch = "arm"
				case "powerpc64":
					arch = "ppc64"
				case "powerpc64le":
					arch = "ppc64le"
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

	others = make([]T, 0, len(work))
	for _, name := range work {
		if checksumsRE.MatchString(name) {
			checksums = append(checksums, fileAssets[name])
		} else {
			others = append(others, fileAssets[name])
		}
	}

	return
}
