// Copyright 2023 Ville Skyttä
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

package util

import (
	"crypto"
	"encoding/hex"
	"fmt"
	"strings"
)

var hashesByName = map[string]crypto.Hash{
	HashName(crypto.MD4):       crypto.MD4,
	HashName(crypto.MD5):       crypto.MD5,
	HashName(crypto.SHA1):      crypto.SHA1,
	HashName(crypto.SHA224):    crypto.SHA224,
	HashName(crypto.SHA256):    crypto.SHA256,
	HashName(crypto.SHA384):    crypto.SHA384,
	HashName(crypto.SHA512):    crypto.SHA512,
	HashName(crypto.RIPEMD160): crypto.RIPEMD160,
}

func HashName(h crypto.Hash) string {
	hn := h.String()
	hn = strings.ToLower(hn)
	hn = strings.ReplaceAll(hn, "-", "")
	return hn
}

// ParseHashFragment prepares a hash corresponding to the given URL fragment string.
// It returns the hash and the digest to check with it.
// If s is empty, 0 is returned as the hash.
func ParseHashFragment(s string) (crypto.Hash, []byte, error) {
	if s == "" {
		return 0, []byte{}, nil
	}
	name, hexHash, found := strings.Cut(s, "-")
	if name == "" || hexHash == "" || !found {
		return 0, nil, fmt.Errorf("invalid fragment format, use hashAlgo-hexDigest")
	}
	digest, err := hex.DecodeString(hexHash)
	if err != nil {
		return 0, nil, err
	}
	var hashType crypto.Hash
	hashType, found = hashesByName[strings.ToLower(name)]
	if !found {
		return 0, nil, fmt.Errorf("no supported hash with name %q", name)
	}
	if !hashType.Available() {
		return 0, nil, fmt.Errorf("hash %s not available", hashType)
	}
	return hashType, digest, nil
}
