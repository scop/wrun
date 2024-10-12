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
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mholt/archiver/v3"
	"github.com/spf13/cobra"

	"github.com/scop/wrun/internal/files"
	"github.com/scop/wrun/internal/hashes"
)

var (
	version       = "devel"
	versionString = ""
)

func init() {
	vs := make([]string, 0, 14)
	vs = append(vs, version)
	if bi, ok := debug.ReadBuildInfo(); ok {
		if bi.GoVersion != "" {
			vs = append(vs, ", built with ", bi.GoVersion)
		}
		for _, bs := range bi.Settings {
			if bs.Key == "GOOS" {
				vs = append(vs, ", for ", bs.Value)
				for _, bs = range bi.Settings {
					if bs.Key == "GOARCH" {
						vs = append(vs, "/", bs.Value)
						break
					}
				}
				break
			}
		}
		for _, bs := range bi.Settings {
			if bs.Key == "vcs" {
				vs = append(vs, ", from ", bs.Value)
				for _, bs = range bi.Settings {
					if bs.Key == "vcs.revision" {
						vs = append(vs, " rev ", bs.Value)
						break
					}
				}
				for _, bs = range bi.Settings {
					if bs.Key == "vcs.time" {
						vs = append(vs, " dated ", bs.Value)
						break
					}
				}
				for _, bs = range bi.Settings {
					if bs.Key == "vcs.modified" {
						if dirty, err := strconv.ParseBool(bs.Value); err == nil && dirty {
							vs = append(vs, " (dirty)")
						}
						break
					}
				}
				break
			}
		}
	}
	versionString = strings.Join(vs, "")
}

type exitStatus = int

const (
	cacheHomeEnvVar           = "WRUN_CACHE_HOME"
	verboseEnvVar             = "WRUN_VERBOSE"
	osArchEnvVar              = "WRUN_OS_ARCH"
	cacheVersion              = "v2"
	cacheDirDigestPlaceholder = "_"
	defaultHTTPTimeout        = 5 * time.Minute

	esSuccess exitStatus = 0
	esError   exitStatus = 1
	esUsage   exitStatus = 2
)

// urlDir gets the cache directory to use for a URL.
func urlDir(u *url.URL, h crypto.Hash, digest []byte) string {
	segs := make([]string, 0, strings.Count(u.Path, "/")+3)
	segs = append(segs, strings.ReplaceAll(u.Host, ":", "_"))
	segs = append(segs, strings.Split(u.Path, "/")...) // Note: filepath.Join later ignores possible empty segments
	if u.RawQuery != "" {
		segs[len(segs)-1] = segs[len(segs)-1] + url.PathEscape("?"+u.RawQuery)
	}
	if h == 0 {
		segs = append(segs, cacheDirDigestPlaceholder)
	} else {
		segs = append(segs, hashes.HashName(h)+"-"+hex.EncodeToString(digest))
	}
	return filepath.Join(segs...)
}

type urlMatch struct {
	pattern string
	url     *url.URL
}

type archiveExePathMatch struct {
	pattern string
	exePath string
}

type rootCmdConfig struct {
	urlMatches            []urlMatch
	archiveExePathMatches []archiveExePathMatch
	dryRun                bool
}

func parseFlags(cfg *rootCmdConfig, urlArgs []string, exePathArgs []string) error {
	for _, s := range urlArgs {
		pattern, ur, found := strings.Cut(s, "=")
		if found {
			if strings.Contains(pattern, "://") {
				ur = s
				pattern = "*/*"
			} else if pattern == "" {
				pattern = "*/*"
			}
		} else {
			ur = pattern
			pattern = "*/*"
		}
		if u, err := url.Parse(ur); err != nil {
			return err
		} else {
			cfg.urlMatches = append(cfg.urlMatches, urlMatch{pattern, u})
		}
	}

	for _, s := range exePathArgs {
		pattern, pth, found := strings.Cut(s, "=")
		if found {
			if pattern == "" {
				pattern = "*/*"
			}
			if pth == "" {
				return fmt.Errorf("missing path in %q", s)
			}
		} else {
			pth = pattern
			pattern = "*/*"
		}
		cfg.archiveExePathMatches = append(cfg.archiveExePathMatches, archiveExePathMatch{pattern, pth})
	}

	return nil
}

func Execute() {

	var urlArgs, exePathArgs []string
	var httpTimeout time.Duration
	w := NewWrun(filepath.Base(os.Args[0]))
	cfg := &rootCmdConfig{}
	rc := esSuccess

	rootCmd := &cobra.Command{
		Use:   w.ProgName + " [flags] -- [executable arguments]",
		Short: w.ProgName + " downloads, caches, and runs executables.",
		Long: fmt.Sprintf(`%s downloads, caches, and runs executables.

OS and architecture matcher arguments for URLs to download and (if applicable) executables within archives can be used to construct command lines that work across multiple operating systems and architectures.

The OS and architecture wrun was built for are matched against the given matchers.
OS and architecture parts of the matcher may be globs.
Order of the matcher arguments is significant: the first match of each is chosen.

As a special case, a matcher argument with no matcher part is treated as if it was given with the matcher */*.
On Windows, .exe is automatically appended to any archive exe path resulting from a */ prefixed match.

URL fragments, if present, are treated as hashAlgo-hexDigest strings, and downloads are checked against them.

The first non-flag argument or -- terminates %s arguments.
Remaining ones are passed to the downloaded executable.

Environment variables:
- %s: cache location, defaults to wrun subdir in the user's cache dir
- %s: override OS/arch for matching
- %s: output verbosity, false decreases, true increases
`, w.ProgName, w.ProgName, cacheHomeEnvVar, osArchEnvVar, verboseEnvVar),
		Args:    cobra.ArbitraryArgs,
		Version: versionString,
		PersistentPreRun: func(_ *cobra.Command, args []string) {
			w.httpClient = &http.Client{
				Timeout: httpTimeout,
			}
		},
		PreRunE: func(_ *cobra.Command, _ []string) error {
			return parseFlags(cfg, urlArgs, exePathArgs)
		},
		Run: func(_ *cobra.Command, args []string) {
			rc = runRoot(w, cfg, args)
		},
	}

	fs := rootCmd.Flags()
	fs.BoolVarP(&cfg.dryRun, "dry-run", "n", false, "dry run, skip execution (but do download/set up cache)")
	fs.StringSliceVarP(&urlArgs, "url", "u", nil, "[OS/arch=]URL matcher (at least one required)")
	if err := rootCmd.MarkFlagRequired("url"); err != nil {
		w.LogBug("mark flag required: %v", err)
	}
	fs.StringSliceVarP(&exePathArgs, "archive-exe-path", "p", nil, "[OS/arch=]path to executable within archive matcher (separator always /, implies archive processing)")
	pfs := rootCmd.PersistentFlags()
	pfs.DurationVarP(&httpTimeout, "http-timeout", "t", defaultHTTPTimeout, "HTTP client timeout")
	if err := rootCmd.RegisterFlagCompletionFunc("http-timeout", cobra.NoFileCompletions); err != nil {
		w.LogBug("register --http-timeout completion: %v", err)
	}

	rootCmd.AddCommand(generateCommand(w))

	if rootCmd.Execute() != nil { // assuming error already printed by cobra
		rc = esUsage
	}
	os.Exit(rc)
}

// selectURL selects a URL for a system from the given matches.
func selectURL(s string, matches []urlMatch) (*url.URL, error) {
	for _, m := range matches {
		match, err := filepath.Match(m.pattern, s)
		if err != nil {
			return nil, err
		}
		if match {
			return m.url, nil
		}
	}
	return nil, nil
}

// selectArchiveExePath selects an archive exe path for a system from the given matches.
func selectArchiveExePath(s string, matches []archiveExePathMatch) (string, error) {
	for _, m := range matches {
		match, err := filepath.Match(m.pattern, s)
		if err != nil {
			return "", err
		}
		if match {
			exePath := m.exePath
			// Auto append .exe to Windows wildcard matches having no filename extension
			if strings.HasPrefix(s, "windows/") && strings.HasPrefix(m.pattern, "*/") && filepath.Ext(exePath) == "" {
				exePath += ".exe"
			}
			return exePath, nil
		}
	}
	return "", nil
}

func resolveCacheDir() (string, error) {
	cacheDir := os.Getenv(cacheHomeEnvVar)
	if cacheDir == "" {
		var err error
		cacheDir, err = os.UserCacheDir()
		if err != nil {
			return "", fmt.Errorf("cache dir: %w", err)
		}
		cacheDir = filepath.Join(cacheDir, "wrun")
	}
	cacheDir = filepath.Join(cacheDir, cacheVersion)
	return cacheDir, nil
}

func runRoot(w *Wrun, cfg *rootCmdConfig, args []string) exitStatus {
	w.LogInfo("%s", versionString)

	// Figure out download URL and exe path in archive

	osArch := os.Getenv(osArchEnvVar)
	if osArch == "" {
		osArch = runtime.GOOS + "/" + runtime.GOARCH
	}
	w.LogInfo("OS/arch: %s", osArch)

	ur, err := selectURL(osArch, cfg.urlMatches)
	if err != nil {
		w.LogError("select URL: %v", err)
		return esUsage // bad pattern
	}
	if ur == nil {
		w.LogError("no URL available for OS/architecture %s", osArch)
		return esError
	}
	w.LogInfo("URL: %s", ur)

	archiveExePath, err := selectArchiveExePath(osArch, cfg.archiveExePathMatches)
	if err != nil {
		w.LogError("select archive exe path: %v", err)
		return esUsage // bad pattern
	}

	// Set up hashing

	hshType, expectedDigest, err := hashes.ParseHashFragment(ur.Fragment)
	if err != nil {
		w.LogError("parse hash fragment: %v", err)
		return esError
	}

	// Set up cache

	var cacheDir string
	cacheDir, err = resolveCacheDir()
	if err != nil {
		w.LogError("cache setup: %v", err)
		return esError
	}
	// Here's hoping we don't hit path too long errors with this implementation anywhere
	ps := make([]string, 0, strings.Count(archiveExePath, "/")+3)
	_, dlBase := path.Split(ur.Path)
	ps = append(ps, cacheDir, urlDir(ur, hshType, expectedDigest), dlBase)
	dlPath := filepath.Join(ps...)
	ps = append(ps, strings.Split(archiveExePath, "/")...)
	exePath := filepath.Join(ps...)
	err = os.MkdirAll(filepath.Dir(exePath), 0o777)
	if err != nil {
		w.LogError("cache setup: %v", err)
		return esError
	}
	w.LogInfo("path to executable: %s", exePath)

	// exec from cache

	exec := func(exe string) error {
		exeArgs := make([]string, len(args)+1)
		exeArgs[0] = exe
		copy(exeArgs[1:], args)
		if cfg.dryRun {
			w.LogInfo("exec (...not, but stat due to dry-run): %v", exeArgs)
			if fi, statErr := os.Stat(exe); statErr != nil {
				return statErr
			} else if !fi.Mode().IsRegular() {
				return fmt.Errorf("not a regular file: %v", exe)
			}
			return nil
		}
		w.LogInfo("exec cached: %v", exeArgs)
		return syscall.Exec(exe, exeArgs, os.Environ())
	}

	if err = exec(exePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			w.LogInfo("exec cached: %v", err)
		} else {
			w.LogWarn("exec cached: %v", err)
		}
	} else if cfg.dryRun {
		return esSuccess
	} else {
		w.LogBug("unreachable; successful non-dry-run cache exec")
	}

	// Set up tempfile for download

	tmpf, cleanUpTempFile, err := w.SetUpTempfile(filepath.Base(dlPath), filepath.Dir(dlPath))
	if err != nil {
		w.LogError("set up tempfile: %v", err)
		return esError
	}
	defer cleanUpTempFile() // Note: defer does not happen if we exec successfully

	// Download and check digest

	resp, err := w.HTTPGet(ur.String())
	if err != nil {
		w.LogError("download: %v", err)
		return esError
	}

	meta := make(map[string]string)
	for _, key := range []string{"ETag", "Last-Modified"} {
		meta[key] = resp.Header.Get(key)
	}

	var hsh hash.Hash
	if hshType != 0 {
		hsh = hshType.New()
	}
	if err = w.Download(resp, tmpf, hsh, expectedDigest); err != nil {
		w.LogError("download: %v", err)
		return esError
	}

	// Move to final location, make executable

	if archiveExePath == "" {
		if err = os.Rename(tmpf.Name(), exePath); err != nil {
			w.LogError("rename tempfile: %v", err)
			return esError
		}
	} else {
		if err = archiver.Unarchive(tmpf.Name(), dlPath); err != nil {
			w.LogError("unarchive: %v", err)
			return esError
		}
	}
	if err = files.MakeExecutable(exePath); err != nil {
		w.LogError("make executable: %v", err)
		return esError
	}

	// Write metadata

	data, err := json.Marshal(meta)
	if err != nil {
		w.LogWarn("encode metadata: %v", err)
	} else if err = os.WriteFile(dlPath+"-metadata.json", data, 0o666); err != nil {
		w.LogWarn("write metadata: %v", err)
	}

	// Execute

	cleanUpTempFile() // Note: deferred cleanup does not happen if we exec successfully
	if err = exec(exePath); err != nil {
		w.LogError("exec: %v", err)
		return esError
	} else if !cfg.dryRun {
		w.LogBug("unreachable; successful non-dry-run exec")
	}

	return esSuccess
}
