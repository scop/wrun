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

package checksums_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	util "github.com/scop/wrun/internal/checksums"
)

func TestChecksums_UnmarshalText(t *testing.T) {
	cases := []struct {
		name        string
		input       []string
		expected    util.Checksums
		errExpected bool
	}{
		{
			name:  "various",
			input: []string{"ff01  filename.txt\nff02 *binfile.bin\n\nff03 one space\n# This is a comment\n", "This is an invalid line\nff04 \t various spaces, trailing preserved \nAnother invalid line"},
			expected: util.Checksums{
				InvalidLines: 2,
				Entries: []util.Entry{
					{Digest: []byte{0xff, 0x01}, BinaryMode: false, Filename: "filename.txt"},
					{Digest: []byte{0xff, 0x02}, BinaryMode: true, Filename: "binfile.bin"},
					{Digest: []byte{0xff, 0x03}, BinaryMode: false, Filename: "one space"},
					{Digest: []byte{0xff, 0x04}, BinaryMode: false, Filename: "various spaces, trailing preserved "},
				},
			},
		},
		{
			name:  "empty",
			input: []string{""},
			expected: util.Checksums{
				InvalidLines: 0,
				Entries:      nil,
			},
			errExpected: true,
		},
		{
			name:  "no filename",
			input: []string{"deadbeef"},
			expected: util.Checksums{
				InvalidLines: 1,
				Entries:      nil,
			},
			errExpected: true,
		},
		{
			name:  "no filename, space",
			input: []string{"deadbeef "},
			expected: util.Checksums{
				InvalidLines: 1,
				Entries:      nil,
			},
			errExpected: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cs := util.Checksums{}
			for _, input := range c.input {
				err := cs.UnmarshalText([]byte(input))
				if c.errExpected {
					assert.Error(t, err) //nolint:testifylint // no require; we want to check checksums return value on error too
				} else {
					assert.NoError(t, err) //nolint:testifylint // no require; we want to check checksums return value on error too
				}
			}
			assert.Equal(t, c.expected, cs)
		})
	}
}
