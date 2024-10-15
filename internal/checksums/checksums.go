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

package checksums

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"errors"
	"strings"
)

type Checksums struct {
	Entries      []Entry
	InvalidLines uint
}

type Entry struct {
	Digest     []byte
	BinaryMode bool
	Filename   string
}

// UnmarshalText reads checksums from data.
// Existing data in c is appended to, not overwritten.
func (c *Checksums) UnmarshalText(text []byte) error {
	s := bufio.NewScanner(bytes.NewReader(text))
	const separators = " \t" // TODO more whitespace like \f, nbsp, etc?
	origN := len(c.Entries)
	for s.Scan() {
		line := s.Text()
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		// strings.Cut, strings.Split* do not work with multiple separators
		// strings.Fields makes a mess with filenames containing spaces
		// fmt.Sscanf also stops at first space in filename
		line = strings.TrimLeft(line, separators)
		ix := strings.IndexAny(line, separators)
		if ix == -1 || ix%2 == 1 {
			c.InvalidLines++

			continue
		}
		digest, err := hex.DecodeString(line[:ix])
		if err != nil {
			c.InvalidLines++

			continue
		}
		entry := Entry{
			Digest: digest,
		}
		filename := strings.TrimLeft(line[ix:], separators)
		if len(filename) != 0 && filename[0] == '*' {
			entry.BinaryMode = true
			entry.Filename = filename[1:]
		} else {
			entry.Filename = filename
		}
		if entry.Filename == "" {
			c.InvalidLines++

			continue
		}
		c.Entries = append(c.Entries, entry)
	}
	if origN == len(c.Entries) {
		return errors.New("no properly formatted checksum lines found")
	}

	return nil
}

func (c *Checksums) MarshalText() ([]byte, error) {
	var buf bytes.Buffer
	for _, entry := range c.Entries {
		_, _ = buf.WriteString(hex.EncodeToString(entry.Digest))
		if entry.BinaryMode {
			_, _ = buf.WriteString(" *")
		} else {
			_, _ = buf.WriteString("  ")
		}
		_, _ = buf.WriteString(entry.Filename)
		_ = buf.WriteByte('\n')
	}

	return buf.Bytes(), nil
}

func (c *Checksums) Get(filename string) []Entry {
	var got []Entry
	for _, entry := range c.Entries {
		if filename == entry.Filename {
			got = append(got, entry)
		}
	}

	return got
}
