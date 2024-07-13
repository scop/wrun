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
	"os"

	"github.com/spf13/cobra"
)

func generateArbitraryPyPIProjectCommand(w *Wrun) *cobra.Command {
	genCmd := &cobra.Command{
		Use:   "pypi PROJECT TOOL [VERSION]",
		Short: "generate wrun command line arguments for a PyPI project",
		Args:  cobra.RangeArgs(2, 3),
		Run: func(_ *cobra.Command, args []string) {
			if err := runGeneratePyPIProject(w, args[0], args[1], args[2:]); err != nil {
				w.LogError("%s", err)
				os.Exit(1)
			}
		},
	}

	return genCmd
}
