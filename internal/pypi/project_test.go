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
		expected pypi.WheelFilenameInfo
		errorMsg string
	}{
		{
			input: "committed-1.0.20-py3-none-win32.whl",
			expected: pypi.WheelFilenameInfo{
				Distribution: "committed",
				Version:      pep440.MustParse("1.0.20"),
				PythonTags:   []string{"py3"},
				ABITags:      []string{"none"},
				PlatformTags: []string{"win32"},
			},
			errorMsg: "",
		},
		{
			input: "shellcheck_py-0.10.0.1-py2.py3-none-manylinux_2_5_x86_64.manylinux1_x86_64.manylinux_2_17_x86_64.manylinux2014_x86_64.whl",
			expected: pypi.WheelFilenameInfo{
				Distribution: "shellcheck_py",
				Version:      pep440.MustParse("0.10.0.1"),
				PythonTags:   []string{"py2", "py3"},
				ABITags:      []string{"none"},
				PlatformTags: []string{"manylinux_2_5_x86_64", "manylinux1_x86_64", "manylinux_2_17_x86_64", "manylinux2014_x86_64"},
			},
			errorMsg: "",
		},
		{
			input:    "invalid-filename.whl",
			expected: pypi.WheelFilenameInfo{},
			errorMsg: "unparseable wheel filename",
		},
		{
			input:    "",
			expected: pypi.WheelFilenameInfo{},
			errorMsg: "",
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			w := pypi.WheelFilenameInfo{}
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
		wni := pypi.WheelFilenameInfo{}
		if err := wni.UnmarshalText([]byte(fn)); err != nil {
			panic(err)
		}
		project.Files = append(project.Files, pypi.SimpleFile{
			Filename:     fn,
			FilenameInfo: wni,
		})
	}

	expected := make([]pep440.Version, 0, len(versions))
	for _, s := range versions {
		expected = append(expected, pep440.MustParse(s))
	}
	slices.Reverse(expected)
	assert.Equal(t, expected, project.Versions())
}
