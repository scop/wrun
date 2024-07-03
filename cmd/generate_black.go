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

package cmd

import (
	"crypto"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/cobra"

	util "github.com/scop/wrun/internal"
)

func generateBlackCommand(w *Wrun) *cobra.Command {
	genCmd := &cobra.Command{
		Use:   "black [VERSION]",
		Short: "generate wrun command line arguments for black",
		Args:  cobra.MaximumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			if err := runGenerateBlack(w, args); err != nil {
				w.LogError("%s", err)
				os.Exit(1)
			}
		},
	}

	return genCmd
}

func runGenerateBlack(w *Wrun, args []string) error {
	var version string
	if len(args) != 0 {
		version = args[0]
	} else {
		vs, err := versionsFromAtom(w, "https://github.com/psf/black/releases.atom")
		if err != nil {
			return fmt.Errorf("get black versions: %s", err)
		}
		version = vs[0]
	}

	files := map[string]string{
		"black_linux":       "linux/amd64",
		"black_macos":       "darwin/amd64",
		"black_windows.exe": "windows/amd64",
	}

	baseURL := "https://github.com/psf/black/releases/download/" + url.PathEscape(version)

	hn, err := util.HashByName(util.HashName(crypto.SHA256))
	if err != nil {
		return err
	}
	hsh := hn.New()
	for fn, osArch := range files {
		u := fmt.Sprintf("%s/%s", baseURL, url.PathEscape(fn))
		resp, err := w.HTTPGet(u)
		if err != nil {
			return err
		}
		if err = w.Download(resp, nil, hn, hsh, nil); err != nil {
			return err
		}
		digest := hsh.Sum(nil)
		hsh.Reset()

		fmt.Printf("--url %s=%s#sha256-%s\n", osArch, u, hex.EncodeToString(digest))
	}

	return nil
}
