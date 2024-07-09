package util_test

import (
	"testing"

	util "github.com/scop/wrun/internal"
	"github.com/stretchr/testify/assert"
)

func TestChecksums_UnmarshalText(t *testing.T) {
	cases := []struct {
		name        string
		input       string
		expected    *util.Checksums
		errExpected bool
	}{
		{
			name:  "various",
			input: "ff01  filename.txt\nff02 *binfile.bin\n\nff03 one space\n# This is a comment\nThis is an invalid line\nff04 \t various spaces, trailing preserved \nAnother invalid line",
			expected: &util.Checksums{
				InvalidLines: 2,
				Entries: []util.ChecksumEntry{
					{Digest: []byte{0xff, 0x01}, BinaryMode: false, Filename: "filename.txt"},
					{Digest: []byte{0xff, 0x02}, BinaryMode: true, Filename: "binfile.bin"},
					{Digest: []byte{0xff, 0x03}, BinaryMode: false, Filename: "one space"},
					{Digest: []byte{0xff, 0x04}, BinaryMode: false, Filename: "various spaces, trailing preserved "},
				},
			},
		},
		{
			name:  "empty",
			input: "",
			expected: &util.Checksums{
				InvalidLines: 0,
				Entries:      nil,
			},
			errExpected: true,
		},
		{
			name:  "no filename",
			input: "deadbeef",
			expected: &util.Checksums{
				InvalidLines: 1,
				Entries:      nil,
			},
			errExpected: true,
		},
		{
			name:  "no filename, space",
			input: "deadbeef ",
			expected: &util.Checksums{
				InvalidLines: 1,
				Entries:      nil,
			},
			errExpected: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cs := &util.Checksums{}
			err := cs.UnmarshalText([]byte(c.input))
			if c.errExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, c.expected, cs)
		})
	}
}
