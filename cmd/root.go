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
	"bytes"
	"crypto"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
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

	wrun "github.com/scop/wrun/internal"
)

var (
	version       = "devel"
	versionString = ""
)

func init() {
	vs := make([]string, 0, 15)
	vs = append(vs, "wrun ", version)
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
		segs = append(segs, wrun.HashName(h)+"-"+hex.EncodeToString(digest))
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

type config struct {
	prog                  string
	urlMatches            []urlMatch
	archiveExePathMatches []archiveExePathMatch
	httpTimeout           time.Duration
	dryRun                bool
}

func parseFlags(cfg *config, urlArgs []string, exePathArgs []string) error {
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
	cfg := config{prog: filepath.Base(os.Args[0])}
	rc := esSuccess

	rootCmd := &cobra.Command{
		Use:   cfg.prog + " [flags] -- [executable arguments]",
		Short: cfg.prog + " downloads, caches, and runs executables.",
		Long: fmt.Sprintf(`%s downloads, caches, and runs executables.

OS and architecture matcher arguments for URLs to download and (if applicable) executables within archives can be used to construct command lines that work across multiple operating systems and architectures.

The OS and architecture wrun was built for are matched against the given matchers.
OS and architecture parts of the matcher may be globs.
Order of the matcher arguments is significant: the first match of each is chosen.

As a special case, a matcher argument with no matcher part is treated as if it was given with the matcher */*.
On Windows, .exe is automatically appended to any archive exe path resulting from a */ prefixed match.

URL fragments, if present, are treated as hashAlgo-hexDigest strings, and downloads are checked against them.

The first non-flag argument or -- terminates %s arguments.
Remaining ones are passed to the downloaded one.

Environment variables:
- %s: cache location, defaults to wrun subdir in the user's cache dir
- %s: override OS/arch for matching
- %s: output verbosity, false decreases, true increases
`, cfg.prog, cfg.prog, cacheHomeEnvVar, osArchEnvVar, verboseEnvVar),
		Args:    cobra.ArbitraryArgs,
		Version: versionString,
		PreRunE: func(_ *cobra.Command, _ []string) error {
			return parseFlags(&cfg, urlArgs, exePathArgs)
		},
		Run: func(_ *cobra.Command, args []string) {
			rc = run(cfg, args)
		},
	}

	fs := rootCmd.Flags()
	fs.BoolVarP(&cfg.dryRun, "dry-run", "n", false, "dry run, skip execution (but do download/set up cache)")
	fs.StringSliceVarP(&urlArgs, "url", "u", nil, "[OS/arch=]URL matcher (at least one required)")
	if err := rootCmd.MarkFlagRequired("url"); err != nil {
		panic(fmt.Sprintf("BUG: mark flag required: %v", err))
	}
	fs.StringSliceVarP(&exePathArgs, "archive-exe-path", "p", nil, "[OS/arch=]path to executable within archive matcher (separator always /, implies archive processing)")
	fs.DurationVarP(&cfg.httpTimeout, "http-timeout", "t", defaultHTTPTimeout, "HTTP client timeout")

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

func run(cfg config, args []string) exitStatus {
	// Set up output

	var verbose *bool
	if s, ok := os.LookupEnv(verboseEnvVar); ok {
		v, _ := strconv.ParseBool(s)
		verbose = &v
	}
	_out := func(w io.Writer, level, format string, a ...any) {
		_, _ = fmt.Fprintf(w, cfg.prog+": "+level+": "+format+"\n", a...)
	}
	infoOut := func(format string, a ...any) {
		if verbose != nil && *verbose {
			_out(os.Stdout, "INFO", format, a...)
		}
	}
	warnOut := func(format string, a ...any) {
		if verbose == nil || *verbose {
			_out(os.Stderr, "WARN", format, a...)
		}
	}
	errorOut := func(format string, a ...any) {
		_out(os.Stderr, "ERROR", format, a...)
	}

	infoOut("%s", versionString)

	// Figure out download URL and exe path in archive

	osArch := os.Getenv(osArchEnvVar)
	if osArch == "" {
		osArch = runtime.GOOS + "/" + runtime.GOARCH
	}
	infoOut("OS/arch: %s", osArch)

	ur, err := selectURL(osArch, cfg.urlMatches)
	if err != nil {
		errorOut("select URL: %v", err)
		return esUsage // bad pattern
	}
	if ur == nil {
		errorOut("no URL available for OS/architecture %s", osArch)
		return esError
	}
	infoOut("URL: %s", ur)

	archiveExePath, err := selectArchiveExePath(osArch, cfg.archiveExePathMatches)
	if err != nil {
		errorOut("select archive exe path: %v", err)
		return esUsage // bad pattern
	}

	// Set up hashing

	hshType, expectedDigest, err := wrun.ParseHashFragment(ur.Fragment)
	if err != nil {
		errorOut("parse hash fragment: %v", err)
		return esError
	}

	// Set up cache

	var cacheDir string
	cacheDir, err = resolveCacheDir()
	if err != nil {
		errorOut("cache setup: %v", err)
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
		errorOut("cache setup: %v", err)
		return esError
	}
	infoOut("path to executable: %s", exePath)

	// exec from cache

	exec := func(exe string) error {
		args := make([]string, len(args)+1)
		args[0] = exe
		copy(args[1:], args)
		if cfg.dryRun {
			infoOut("exec (...not, stat due to dry-run): %v", args)
			if fi, statErr := os.Stat(exe); statErr != nil {
				return statErr
			} else if !fi.Mode().IsRegular() {
				return fmt.Errorf("not a regular file: %v", exe)
			}
			return nil
		}
		infoOut("exec: %v", args)
		return syscall.Exec(exe, args, os.Environ())
	}

	if err = exec(exePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			infoOut("exec cached: %v", err)
		} else {
			warnOut("exec cached: %v", err)
		}
	} else if cfg.dryRun {
		return esSuccess
	} else {
		panic("BUG: unreachable; successful non-dry-run cache exec")
	}

	// Set up tempfile for download

	// Use temp filename _prefix_, archiver recognizes by filename extension
	tmpf, err := os.CreateTemp(filepath.Dir(dlPath), "tmp*-"+filepath.Base(dlPath))
	if err != nil {
		errorOut("set up tempfile: %v", err)
		return esError
	}
	cleanUpTempFile := func() {
		if closeErr := tmpf.Close(); closeErr != nil && !errors.Is(closeErr, os.ErrClosed) {
			warnOut("close tempfile: %v", closeErr)
		}
		if rmErr := os.Remove(tmpf.Name()); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) {
			warnOut("remove tempfile: %v", rmErr)
		}
	}
	defer cleanUpTempFile() // Note: does not happen if we exec successfully

	// Download

	hc := http.Client{
		Timeout: cfg.httpTimeout,
	}
	req, err := http.NewRequest(http.MethodGet, ur.String(), nil)
	if err != nil {
		errorOut("prepare request: %v", err)
		return esError
	}
	req.Header.Set("User-Agent", cfg.prog+"/"+version)
	// TODO if no checksum, do conditional get: If-None-Match, If-Modified-Since?

	resp, err := hc.Do(req)
	if err != nil {
		errorOut("get %s: %v", ur.String(), err)
		return esError
	}

	var (
		hsh hash.Hash
		w   io.Writer
	)
	if hshType == 0 {
		w = tmpf
	} else {
		hsh = hshType.New()
		w = io.MultiWriter(hsh, tmpf)
	}

	meta := make(map[string]string)
	for _, key := range []string{"ETag", "Last-Modified"} {
		meta[key] = resp.Header.Get(key)
	}

	n, err := io.Copy(w, resp.Body)
	if closeErr := resp.Body.Close(); closeErr != nil {
		warnOut("close http response: %v", closeErr)
	}
	if err != nil {
		errorOut("download: %v", err)
		return esError
	}
	if err = tmpf.Close(); err != nil {
		errorOut("close tempfile: %v", err)
		return esError
	}
	infoOut("downloaded %d bytes", n)

	// Check digest

	if hsh != nil {
		gotDigest := hsh.Sum(nil)
		if bytes.Equal(gotDigest, expectedDigest) {
			infoOut("%s digest match: %s", hshType, hex.EncodeToString(expectedDigest))
		} else {
			errorOut("%s digest mismatch: got %s, expected %s", hex.EncodeToString(gotDigest), hex.EncodeToString(expectedDigest))
			return esError
		}
	}

	// Move to final location, make executable

	if archiveExePath == "" {
		if err = os.Rename(tmpf.Name(), exePath); err != nil {
			errorOut("rename tempfile: %v", err)
			return esError
		}
	} else {
		var archiveName string
		if strings.HasSuffix(tmpf.Name(), ".whl") { // Need to rename to .zip for archiver
			archiveName = strings.TrimSuffix(tmpf.Name(), ".whl") + ".zip"
			if err = os.Symlink(filepath.Base(tmpf.Name()), archiveName); err != nil { // Failure if new name exists is desirable
				errorOut("symlink tempfile: %v", err)
			}
		} else {
			archiveName = tmpf.Name()
		}
		err = archiver.Unarchive(archiveName, dlPath)
		if archiveName != tmpf.Name() {
			if rmErr := os.Remove(archiveName); rmErr != nil {
				warnOut("remove tempfile symlink: %v", rmErr)
			}
		}
		if err != nil {
			errorOut("unarchive: %v", err)
			return esError
		}
	}
	if err = wrun.MakeExecutable(exePath); err != nil {
		errorOut("make executable: %v", err)
		return esError
	}

	// Write metadata

	data, err := json.Marshal(meta)
	if err != nil {
		warnOut("encode metadata: %v", err)
	} else if err = os.WriteFile(dlPath+"-metadata.json", data, 0o666); err != nil {
		warnOut("write metadata: %v", err)
	}

	// Execute

	cleanUpTempFile() // Note: deferred cleanup does not happen if we exec successfully
	if err = exec(exePath); err != nil {
		errorOut("exec: %v", err)
		return esError
	} else if !cfg.dryRun {
		panic("BUG: unreachable; successful non-dry-run exec")
	}

	return esSuccess
}
