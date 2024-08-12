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

package files_test

import (
	"regexp"
	"testing"

	"github.com/scop/wrun/internal/files"
	"github.com/stretchr/testify/assert"
)

func TestCategorize_Basic(t *testing.T) {
	for _, path := range []string{"https://github.com/scop/wrun/releases/v1.0.0/", ""} {

		fileAssets := map[string]string{
			path + "checksums.txt":                          "checksums",
			path + "example-1.0.0-darwin-arm64.zip":         "darwin-arm64-zip",
			path + "example-1.0.0-darwin-arm64.tar.gz":      "darwin-arm64",
			path + "example-1.0.0-linux-amd64.deb":          "deb",
			path + "example-1.0.0-linux-x86_64.tar.gz":      "linux-amd64",
			path + "example-1.0.0-linux-armv6.tar.gz":       "linux-armv6-ignored", // expected ignored, armv7 takes precedence
			path + "example-1.0.0-linux-armv7.tar.gz":       "linux-armv7",
			path + "example-1.0.0-windows-amd64.zip":        "windows-amd64",
			path + "example-1.0.0-windows-amd64.zip.sha256": "windows-amd64-sha256",
			path + "example-foo.zip":                        "override-foo",
			path + "example.linux.loong64.tar.xz":           "linux-loong64",
			path + "example-aarch64-unknown-linux-musl.zip": "linux-arm64",
			path + "example-aarch64-unknown-linux-gnu.zip":  "linux-arm64-ignored", // expected ignored, musl takes precedence
			path + "example-other.tar.bz2":                  "example-other",
		}
		overrides := map[string]*regexp.Regexp{
			"linux/s390x": regexp.MustCompile(`-foo\.zip$`),
		}
		expectedPreferred := map[string]string{
			"darwin/arm64":  "darwin-arm64",
			"linux/amd64":   "linux-amd64",
			"linux/arm":     "linux-armv7",
			"linux/arm64":   "linux-arm64",
			"linux/loong64": "linux-loong64",
			"linux/s390x":   "override-foo",
			"windows/amd64": "windows-amd64",
		}
		expectedSums := []string{
			"checksums",
			"windows-amd64-sha256",
		}
		expectedOthers := []string{
			"deb",
			"example-other",
		}

		preferred, sums, others := files.Categorize(fileAssets, overrides)
		assert.Equal(t, expectedPreferred, preferred, "preferred assets")
		assert.ElementsMatch(t, expectedSums, sums, "checksum assets")
		assert.ElementsMatch(t, expectedOthers, others, "other assets")
	}
}

func TestCategorize_NonArchive(t *testing.T) {
	for _, path := range []string{"https://github.com/mvdan/sh/releases/download/v3.8.0/", ""} {
		fileAssets := map[string]string{
			path + "sha256sums.txt":                 "checksums",
			path + "shfmt_v3.8.0_darwin_amd64":      "darwin-amd64",
			path + "shfmt_v3.8.0_darwin_arm64":      "darwin-arm64",
			path + "shfmt_v3.8.0_linux_386":         "linux-386",
			path + "shfmt_v3.8.0_linux_amd64":       "linux-amd64",
			path + "shfmt_v3.8.0_linux_arm":         "linux-arm",
			path + "shfmt_v3.8.0_linux_arm64":       "linux-arm64",
			path + "shfmt_v3.8.0_windows_386.exe":   "windows-386",
			path + "shfmt_v3.8.0_windows_amd64.exe": "windows-amd64",
		}
		expectedPreferred := map[string]string{
			"darwin/amd64":  "darwin-amd64",
			"darwin/arm64":  "darwin-arm64",
			"linux/386":     "linux-386",
			"linux/amd64":   "linux-amd64",
			"linux/arm":     "linux-arm",
			"linux/arm64":   "linux-arm64",
			"windows/386":   "windows-386",
			"windows/amd64": "windows-amd64",
		}
		expectedSums := []string{
			"checksums",
		}
		expectedOthers := []string{}

		preferred, sums, others := files.Categorize(fileAssets, nil)
		assert.Equal(t, expectedPreferred, preferred, "preferred assets")
		assert.ElementsMatch(t, expectedSums, sums, "checksum assets")
		assert.ElementsMatch(t, expectedOthers, others, "other assets")
	}
}
