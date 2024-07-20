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

package cmd

import (
	"archive/tar"
	"fmt"
	"strings"

	"github.com/klauspost/compress/zip"
	"github.com/mholt/archiver/v3"
	"github.com/scop/wrun/internal/files"
	"github.com/spf13/cobra"
)

func generateCommand(w *Wrun) *cobra.Command {
	genCmd := &cobra.Command{
		Use:   "generate",
		Short: "generate wrun command line arguments for various tools",
		Args:  cobra.NoArgs,
	}
	genCmd.AddCommand(
		generateArbitraryGitHubProjectCommand(w),
		generateArbitraryPyPIProjectCommand(w),
		generateBlackCommand(w),
		generateCommittedCommand(w),
		generateDprintCommand(w),
		generateGolangciLintCommand(w),
		generateHadolintCommand(w),
		generateRuffCommand(w),
		generateShfmtCommand(w),
		generateTerraformCommand(w),
		generateTflintCommand(w),
		generateTrivyommand(w),
		generateTyposCommand(w),
		generateVacuumCommand(w),
	)
	return genCmd
}

func findToolInArchive(filename, toolExe string) (path string, err error) {
	// TODO maybe if there's just one file in the archive, use it despite of tool name match?
	err = archiver.Walk(filename, func(f archiver.File) error {
		if !f.IsDir() && f.Name() == toolExe {
			// Need to look in the archive type specific Header for the full path
			switch fh := f.Header.(type) {
			case *tar.Header:
				path = fh.Name
			case zip.FileHeader:
				path = fh.Name
			default:
				return fmt.Errorf("unsupported file header: %T", f.Header)
			}
			// Prefer executables over others
			if files.HasExecutablePerms(f) {
				return archiver.ErrStopWalk
			}
		}
		return nil
	})
	if err != nil {
		err = fmt.Errorf("walk archive: %w", err)
	}
	return
}

func generateExePathArgs(osArchExePaths map[string]string) []string {
	ret := make([]string, 0, len(osArchExePaths))
	if len(osArchExePaths) != 0 {
		// Simplify output if path is the same in all archives (and if the tool was found in all of them).
		// If a tool was not found in some of the archives, warn about it and ignore but proceed. Output the url arg though in the warning so it can be fixed up manually by the user.
		var prevExePath string
		sameExePath := true
		for osArch, exePath := range osArchExePaths {
			ret = append(ret, osArch+"="+exePath)
			if strings.HasPrefix(osArch, "windows/") && strings.HasSuffix(strings.ToLower(exePath), ".exe") {
				exePath = exePath[:len(exePath)-len(".exe")]
			}
			if prevExePath != "" && prevExePath != exePath {
				sameExePath = false
			}
			prevExePath = exePath
		}
		if sameExePath {
			ret = []string{prevExePath}
		}
	}
	return ret
}
