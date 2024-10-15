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
	"regexp"

	"github.com/spf13/cobra"
)

func generateBlackCommand(w *Wrun) *cobra.Command {
	overrides := map[string]*regexp.Regexp{
		"darwin/amd64":  regexp.MustCompile(`_macos$`),
		"linux/amd64":   regexp.MustCompile(`_linux$`),
		"windows/amd64": regexp.MustCompile(`_windows\.exe$`),
	}

	return generateGitHubProjectCommand(w, "psf", "black", "black", overrides)
}
