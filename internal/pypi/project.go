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
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"

	pep440 "github.com/aquasecurity/go-pep440-version"
)

// https://packaging.python.org/en/latest/specifications/simple-repository-api/#json-based-simple-api-for-python-package-indexes

type SimpleProject struct {
	Name  string       `json:"name"`
	Files []SimpleFile `json:"files"`

	// There is a "versions" array in version 1.1 of the API responses that contains only the version strings.
	// However, we need to parse filenames of all files anyway to determine files belonging to each version,
	// so we do not have need for the separately available ones.
	// If we sometimes do, check the status of https://github.com/aquasecurity/go-pep440-version/pull/3
}

type Yanked string

func (y *Yanked) UnmarshalJSON(data []byte) error {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if r, ok := raw.(bool); ok {
		if r {
			// Should not happen per the spec, but just in case
			*y = "wrun: no yanked reason"
		} else {
			*y = ""
		}
	} else if r, ok := raw.(string); ok {
		*y = Yanked(r)
	} else {
		return fmt.Errorf("invalid type: %T", raw)
	}
	return nil
}

type Filename struct {
	string
	Info FilenameInfo
}

func NewFilename(filename string) Filename {
	f := Filename{string: filename}
	_ = f.Info.UnmarshalText([]byte(filename))
	return f
}

func (f *Filename) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*f = NewFilename(s)
	return nil
}

type SimpleFile struct {
	Filename Filename         `json:"filename"`
	URL      string           `json:"url"`
	Hashes   SimpleFileHashes `json:"hashes"`
	// Yanked is the reason for yanking, or empty if not yanked.
	// This deviates from the PyPI simple API where the reason is a non-empty string when yanked, and boolean false when not.
	Yanked Yanked `json:"yanked"`
}

type SimpleFileHashes struct {
	SHA256 string `json:"sha256"`
}

// https://packaging.python.org/en/latest/specifications/binary-distribution-format/#file-name-convention
var wheelFilenameRE = regexp.MustCompile(`^` +
	`(?P<distribution>[^-]+)` +
	`-(?P<version>[^-]+)` +
	`(?:-(?P<build_tag>[0-9][^-]*))?` +
	`-(?P<python_tags>[^.-]+(?:\.[^.-]+)*)` +
	`-(?P<abi_tags>[^.-]+(?:\.[^.-]+)*)` +
	`-(?P<platform_tags>[^.-]+(?:\.[^.-]+)*)` +
	`\.whl$`)

// https://packaging.python.org/en/latest/specifications/source-distribution-format/#source-distribution-file-name
var sdistFilenameRE = regexp.MustCompile(`^` +
	`(?P<distribution>[^-]+)` +
	`-(?P<version>[^-]+)` +
	// Formats other than .tar.gz are obsolete,
	// and I'm (scop) not sure offhand if they can be or could ever have been uploaded to PyPI in the first place,
	// but just in case as this does not seem prone to false positives, https://docs.python.org/3.11/distutils/sourcedist.html
	`\.(?:tar(?:\.(?:bz2|[gx]z|Z))?|zip)$`)

type FilenameInfo struct {
	Distribution string
	Version      pep440.Version
	BuildTag     string
	PythonTags   []string
	ABITags      []string
	// https://packaging.python.org/en/latest/specifications/platform-compatibility-tags/#platform-tag
	PlatformTags         []string
	IsBinaryDistribution bool
}

func (f *FilenameInfo) UnmarshalText(data []byte) error {
	if m := wheelFilenameRE.FindStringSubmatch(string(data)); m != nil {
		info := FilenameInfo{IsBinaryDistribution: true}
		for i, name := range wheelFilenameRE.SubexpNames() {
			switch name {
			case "":
				// whole expression
			case "distribution":
				info.Distribution = m[i]
			case "version":
				var err error
				if info.Version, err = pep440.Parse(m[i]); err != nil {
					return err
				}
			case "build_tag":
				info.BuildTag = m[i]
			case "python_tags":
				info.PythonTags = strings.Split(m[i], ".")
			case "abi_tags":
				info.ABITags = strings.Split(m[i], ".")
			case "platform_tags":
				info.PlatformTags = strings.Split(m[i], ".")
			default:
				panic("wrun: BUG : unhandled wheel subexpression name: " + name)
			}
		}
		*f = info
		return nil
	}
	if m := sdistFilenameRE.FindStringSubmatch(string(data)); m != nil {
		info := FilenameInfo{IsBinaryDistribution: false}
		for i, name := range sdistFilenameRE.SubexpNames() {
			switch name {
			case "":
				// whole expression
			case "distribution":
				info.Distribution = m[i]
			case "version":
				var err error
				if info.Version, err = pep440.Parse(m[i]); err != nil {
					return err
				}
			default:
				panic("wrun: BUG : unhandled sdist subexpression name: " + name)
			}
		}
		*f = info
		return nil
	}
	return fmt.Errorf("unparseable filename: %q", string(data))
}

// ValidVersions returns versions of the project, sorted in newest to oldest order.
// Non-PEP-440 versions are ignored, as are ones containing no files or yanked files only.
func (p SimpleProject) ValidVersions() []pep440.Version {
	pvMap := make(map[string]pep440.Version, len(p.Files)/2)
	for _, file := range p.Files {
		if file.Yanked != "" {
			continue
		}
		version := file.Filename.Info.Version
		if !reflect.ValueOf(version).IsZero() { // https://github.com/aquasecurity/go-pep440-version/pull/4
			pvMap[version.String()] = version
		}
	}
	pepVersions := make([]pep440.Version, 0, len(pvMap))
	for _, pv := range pvMap {
		pepVersions = append(pepVersions, pv)
	}
	slices.SortFunc(pepVersions, func(a, b pep440.Version) int {
		return -a.Compare(b)
	})
	return pepVersions
}

var osArchPlatformTags = []map[string]string{
	{
		"darwin/amd64":  "macosx_*_x86_64",
		"darwin/arm64":  "macosx_*_arm64",
		"linux/386":     "musllinux_*_i686",
		"linux/amd64":   "musllinux_*_x86_64",
		"linux/arm":     "musllinux_*_armv7l",
		"linux/arm64":   "musllinux_*_aarch64",
		"windows/386":   "win32",
		"windows/amd64": "win_amd64",
		"windows/arm64": "win_arm64",
	},
	{
		"darwin/amd64":  "macosx_*_universal2",
		"darwin/arm64":  "macosx_*_universal2",
		"linux/386":     "manylinux*_i686",
		"linux/amd64":   "manylinux*_x86_64",
		"linux/arm":     "manylinux*_armv7l",
		"linux/arm64":   "manylinux*_aarch64",
		"linux/ppc64":   "manylinux*_ppc64",
		"linux/ppc64le": "manylinux*_ppc64le",
		"linux/s390x":   "manylinux*_s390x",
	},
	{
		"linux/arm": "linux_armv6l",
	},
}

func (p SimpleProject) PreferredOsArchSimpleFiles(version string) (osArchPreferred map[string]SimpleFile, others []SimpleFile) {
	osArchPreferred = make(map[string]SimpleFile, len(p.Files))
	for _, file := range p.Files {
		if !file.Filename.Info.IsBinaryDistribution || file.Yanked != "" || file.Filename.Info.Version.String() != version {
			continue
		}

		gotMatch := false
		for _, oapt := range osArchPlatformTags {
			for osArch, pattern := range oapt {
				for _, pt := range file.Filename.Info.PlatformTags {
					// Try match first before existing osArch lookup for proper tracking of others
					if m, err := filepath.Match(pattern, pt); err == nil && m {
						if _, found := osArchPreferred[osArch]; !found {
							osArchPreferred[osArch] = file
						}
						gotMatch = true
					}
				}
			}
		}
		if !gotMatch {
			others = append(others, file)
		}
	}
	return osArchPreferred, others
}
