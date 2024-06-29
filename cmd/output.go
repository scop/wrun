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
	"fmt"
	"os"
	"strconv"
)

type Level = string

const (
	levelBug   Level = "BUG"
	levelError Level = "ERR"
	levelWarn  Level = "WARN"
	levelInfo  Level = "INFO"
)

var levelFormat = fmt.Sprintf("%%%ds", len(levelInfo)) // should be longest of them

func nullOutput(_ string, _ ...any) {}

// outputFn returns a function that logs messages based on level and verbosity configuration.
// The returned function takes a format string and optional arguments, and logs the formatted message
// to the standard error stream. If the level is levelBug, the function will panic with the logged message.
func outputFn(cfg config, level Level) func(format string, args ...any) {
	var verbose *bool
	if s, ok := os.LookupEnv(verboseEnvVar); ok {
		v, _ := strconv.ParseBool(s)
		verbose = &v
	}

	switch {
	case level == levelInfo && (verbose != nil && *verbose):
		fallthrough
	case level == levelWarn && (verbose == nil || *verbose):
		fallthrough
	case level == levelError || level == levelBug:
		return func(format string, args ...interface{}) {
			s := fmt.Sprintf(cfg.prog+": "+fmt.Sprintf(levelFormat, level)+": "+format, args...)
			if level == levelBug {
				panic(s)
			}
			fmt.Fprintln(os.Stderr, s)
		}
	}
	return nullOutput
}
