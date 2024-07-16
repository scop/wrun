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
	"testing"

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
				Version:      "1.0.20",
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
				Version:      "0.10.0.1",
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
