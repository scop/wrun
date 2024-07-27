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

package pypi_test

import (
	"fmt"
	"slices"
	"testing"

	pep440 "github.com/aquasecurity/go-pep440-version"
	"github.com/scop/wrun/internal/pypi"
	"github.com/stretchr/testify/assert"
)

func TestUnmarshalText(t *testing.T) {
	tests := []struct {
		input    string
		expected pypi.FilenameInfo
		errorMsg string
	}{
		{
			input: "committed-1.0.20-py3-none-win32.whl",
			expected: pypi.FilenameInfo{
				Distribution:         "committed",
				Version:              pep440.MustParse("1.0.20"),
				PythonTags:           []string{"py3"},
				ABITags:              []string{"none"},
				PlatformTags:         []string{"win32"},
				IsBinaryDistribution: true,
			},
			errorMsg: "",
		},
		{
			input: "shellcheck_py-0.10.0.1-py2.py3-none-manylinux_2_5_x86_64.manylinux1_x86_64.manylinux_2_17_x86_64.manylinux2014_x86_64.whl",
			expected: pypi.FilenameInfo{
				Distribution:         "shellcheck_py",
				Version:              pep440.MustParse("0.10.0.1"),
				PythonTags:           []string{"py2", "py3"},
				ABITags:              []string{"none"},
				PlatformTags:         []string{"manylinux_2_5_x86_64", "manylinux1_x86_64", "manylinux_2_17_x86_64", "manylinux2014_x86_64"},
				IsBinaryDistribution: true,
			},
			errorMsg: "",
		},
		{
			input:    "invalid-filename.whl",
			expected: pypi.FilenameInfo{},
			errorMsg: "unparseable filename",
		},
		{
			input: "distro-1.42.42.tar.gz",
			expected: pypi.FilenameInfo{
				Distribution:         "distro",
				Version:              pep440.MustParse("1.42.42"),
				IsBinaryDistribution: false,
			},
			errorMsg: "",
		},
		{
			input:    "",
			expected: pypi.FilenameInfo{},
			errorMsg: "unparseable filename",
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			w := pypi.FilenameInfo{}
			err := w.UnmarshalText([]byte(test.input))
			if test.errorMsg == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, test.errorMsg)
			}
			assert.Equal(t, test.expected, w)
		})
	}
}

func TestVersions(t *testing.T) {
	project := pypi.SimpleProject{
		Name: "test",
	}
	// https://packaging.python.org/en/latest/specifications/version-specifiers/#summary-of-permitted-suffixes-and-relative-ordering
	// Note: oldest to latest order expected here
	versions := []string{
		"1.dev0",
		"1.0.dev456",
		"1.0a1",
		"1.0a2.dev456",
		"1.0a12.dev456",
		"1.0a12",
		"1.0b1.dev456",
		"1.0b2",
		"1.0b2.post345.dev456",
		"1.0b2.post345",
		"1.0rc1.dev456",
		"1.0rc1",
		"1.0",
		// Versions with local version identifiers (plus something) cannot be uploaded in PyPI,
		// ignoring them here as there are ambiguities with representing them in wheel filenames:
		// https://github.com/pypa/pip/issues/9628
		// "1.0+abc.5",
		// "1.0+abc.7",
		// "1.0+5",
		"1.0.post456.dev34",
		"1.0.post456",
		"1.0.15",
		"1.1.dev1",
	}
	for _, s := range versions {
		fn := fmt.Sprintf("%s-%s-py3-none-manylinux_2_17_aarch64.manylinux2014_aarch64.whl", project.Name, s)
		project.Files = append(project.Files, pypi.SimpleFile{Filename: pypi.NewFilename(fn)})
	}

	expected := make([]pep440.Version, 0, len(versions))
	for _, s := range versions {
		expected = append(expected, pep440.MustParse(s))
	}
	slices.Reverse(expected)
	assert.Equal(t, expected, project.ValidVersions())
}

func TestPreferredOsArchSimpleFiles(t *testing.T) {
	p := pypi.SimpleProject{
		Name: "test",
	}
	version := "1.1.1"
	expectedPreferred := map[string]pypi.SimpleFile{
		"darwin/amd64":  {Filename: pypi.NewFilename(fmt.Sprintf("%s-%s-py3-none-macosx_10_9_universal2.whl", p.Name, version))},
		"darwin/arm64":  {Filename: pypi.NewFilename(fmt.Sprintf("%s-%s-py3-none-macosx_10_9_universal2.whl", p.Name, version))},
		"linux/386":     {Filename: pypi.NewFilename(fmt.Sprintf("%s-%s-py3-none-manylinux_2_17_i686.manylinux2014_i686.whl", p.Name, version))},
		"linux/amd64":   {Filename: pypi.NewFilename(fmt.Sprintf("%s-%s-py3-none-musllinux_1_2_x86_64.whl", p.Name, version))},
		"linux/arm64":   {Filename: pypi.NewFilename(fmt.Sprintf("%s-%s-py3-none-musllinux_1_2_aarch64.whl", p.Name, version))},
		"windows/386":   {Filename: pypi.NewFilename(fmt.Sprintf("%s-%s-py3-none-win32.whl", p.Name, version))},
		"windows/amd64": {Filename: pypi.NewFilename(fmt.Sprintf("%s-%s-py3-none-win_amd64.whl", p.Name, version))},
	}
	expectedIgnored := []pypi.SimpleFile{
		{Filename: pypi.NewFilename(fmt.Sprintf("%s-%s-py3-none-manylinux_2_17_aarch64.manylinux2014_aarch64.whl", p.Name, version))},
		{Filename: pypi.NewFilename(fmt.Sprintf("%s-%s-py3-none-manylinux_2_17_x86_64.manylinux2014_x86_64.whl", p.Name, version))},
	}

	for _, sf := range expectedPreferred {
		p.Files = append(p.Files, sf)
	}
	p.Files = append(p.Files, expectedIgnored...)

	osArchPreferred, others := p.PreferredOsArchSimpleFiles(version)

	assert.Equal(t, expectedPreferred, osArchPreferred)
	assert.Empty(t, others) // expectedIgnored NOT to be included here, but silently skipped
}
