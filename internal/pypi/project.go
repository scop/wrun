// Copyright 2024 Ville Skyttä
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package pypi

import (
	"errors"
	"path/filepath"
	"regexp"
	"strings"
)

type SimpleProject struct {
	Name  string       `json:"name"`
	Files []SimpleFile `json:"files"`
}

type SimpleFile struct {
	Filename     string            `json:"filename"`
	URL          string            `json:"url"`
	Hashes       SimpleFileHashes  `json:"hashes"`
	Yanked       bool              `json:"yanked"`
	FilenameInfo WheelFilenameInfo `json:"-"`
}

type SimpleFileHashes struct {
	SHA256 string `json:"sha256"`
}

// https://packaging.python.org/en/latest/specifications/binary-distribution-format/#file-name-convention
var wheelFilenameRE = regexp.MustCompile(`^` +
	`(?P<distribution>[^-]+)` +
	`-(?P<version>[^-]+)` +
	`(?:-(?P<build_tag>[0-9][^-]*))?` +
	`-(?P<python_tag>[^.-]+(?:\.[^.-]+)*)` +
	`-(?P<abi_tag>[^.-]+(?:\.[^.-]+)*)` +
	`-(?P<platform_tag>[^.-]+(?:\.[^.-]+)*)` +
	`\.whl$`)

type WheelFilenameInfo struct {
	Distribution string
	Version      string
	BuildTag     string
	PythonTags   []string
	ABITags      []string
	// https://packaging.python.org/en/latest/specifications/platform-compatibility-tags/#platform-tag
	PlatformTags []string
}

func (w *WheelFilenameInfo) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		*w = WheelFilenameInfo{}
		return nil
	}
	if m := wheelFilenameRE.FindStringSubmatch(string(data)); m != nil {
		info := WheelFilenameInfo{}
		for i, name := range wheelFilenameRE.SubexpNames() {
			switch name {
			case "":
				// whole expression
			case "distribution":
				info.Distribution = m[i]
			case "version":
				info.Version = m[i]
			case "build_tag":
				info.BuildTag = m[i]
			case "python_tag":
				info.PythonTags = strings.Split(m[i], ".")
			case "abi_tag":
				info.ABITags = strings.Split(m[i], ".")
			case "platform_tag":
				info.PlatformTags = strings.Split(m[i], ".")
			default:
				panic("unhandled subexpression name: " + name)
			}
		}
		*w = info
		return nil
	}
	return errors.New("unparseable wheel filename")
}

func (p SimpleProject) Versions() []string {
	versions := make(map[string]any, len(p.Files)/2)
	for _, file := range p.Files {
		// TODO make this happen on JSON unmarshal or smth
		if err := file.FilenameInfo.UnmarshalText([]byte(file.Filename)); err != nil {
			continue
		}
		versions[file.FilenameInfo.Version] = struct{}{}
	}
	result := make([]string, 0, len(versions))
	for k := range versions {
		result = append(result, k)
	}
	// TODO PEP 440 sort descending, https://github.com/aquasecurity/go-pep440-version

	return result
}

var osArchPlatformTags = map[string]string{
	"darwin/amd64":  "macosx_*_x86_64",
	"darwin/arm64":  "macosx_*_arm64",
	"linux/386":     "musllinux_*_i686",
	"linux/amd64":   "musllinux_*_x86_64",
	"linux/arm":     "musllinux_*_armv7l",
	"linux/arm64":   "musllinux_*_aarch64",
	"windows/386":   "win32",
	"windows/amd64": "win_amd64",
	"windows/arm64": "win_arm64",
}

var osArchSecondaryPlatformTags = map[string]string{
	"linux/386":     "manylinux*_i686",
	"linux/amd64":   "manylinux*_x86_64",
	"linux/arm":     "manylinux*_armv7l",
	"linux/arm64":   "manylinux*_aarch64",
	"linux/ppc64":   "manylinux*_ppc64",
	"linux/ppc64le": "manylinux*_ppc64le",
	"linux/s390x":   "manylinux*_s390x",
}

func (p SimpleProject) PreferredOsArchSimpleFiles(version string) (hits map[string]SimpleFile, misses []SimpleFile) {
	hits = make(map[string]SimpleFile, len(p.Files))
files:
	for _, file := range p.Files {
		// TODO make this happen on JSON unmarshal or smth
		if err := file.FilenameInfo.UnmarshalText([]byte(file.Filename)); err != nil {
			continue
		}
		if file.Yanked || file.FilenameInfo.Version != version {
			continue
		}

		for osArch, pattern := range osArchPlatformTags {
			for _, pt := range file.FilenameInfo.PlatformTags {
				if m, err := filepath.Match(pattern, pt); err == nil && m {
					hits[osArch] = file
					continue files
				}
			}
		}
		for osArch, pattern := range osArchSecondaryPlatformTags {
			// Try match first before existing osArch lookup for proper tracking of misses
			for _, pt := range file.FilenameInfo.PlatformTags {
				if m, err := filepath.Match(pattern, pt); err == nil && m {
					if _, found := hits[osArch]; !found {
						hits[osArch] = file
					}
					continue files
				}
			}
		}
		misses = append(misses, file)
	}
	return hits, misses
}
