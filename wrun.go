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

package main

import (
	"bytes"
	"crypto"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/adrg/xdg"
)

var (
	version = "dev"
)

const (
	cacheVersion         = "v1"
	hexDigestPlaceholder = "_"
)

var hashes = map[int]crypto.Hash{
	crypto.MD5.Size():    crypto.MD5,
	crypto.SHA1.Size():   crypto.SHA1,
	crypto.SHA224.Size(): crypto.SHA224,
	crypto.SHA256.Size(): crypto.SHA256,
	crypto.SHA384.Size(): crypto.SHA384,
	crypto.SHA512.Size(): crypto.SHA512,
}

// prepareHash prepares a hash corresponding to the given digest string.
// It returns the hash and the digest to check with it.
// If s is empty, 0 is returned as the hash.
func prepareHash(s string) (crypto.Hash, []byte, error) {
	if s == "" {
		return 0, []byte{}, nil
	}
	var (
		digest   []byte
		hashType crypto.Hash
	)
	digest, err := hex.DecodeString(s)
	if err != nil {
		return 0, nil, err
	}
	var found bool
	hashType, found = hashes[len(digest)] // TODO could support something more elaborate, e.g. `{algo}hexdigest` or `algo-hexdigest` or `algo-base64digest` (SRI, https://w3c.github.io/webappsec-subresource-integrity/)
	if !found {
		return 0, nil, fmt.Errorf("no supported hash with digest size %d", len(digest))
	}
	if !hashType.Available() {
		return 0, nil, fmt.Errorf("hash %s not available", hashType)
	}
	return hashType, digest, nil
}

// urlDir gets the cache directory to use for a URL.
func urlDir(u *url.URL) string {
	segs := make([]string, 0, strings.Count(u.Path, "/")+3)
	segs = append(segs, strings.ReplaceAll(u.Host, ":", "_"))
	segs = append(segs, strings.Split(u.Path, "/")...) // Note: filepath.Join later ignores possible empty segments
	if u.RawQuery != "" {
		segs[len(segs)-1] = segs[len(segs)-1] + url.PathEscape("?"+u.RawQuery)
	}
	if u.Fragment == "" {
		segs = append(segs, hexDigestPlaceholder)
	} else {
		segs = append(segs, u.Fragment)
	}
	return filepath.Join(segs...)
}

type urlMatch struct {
	pattern string
	url     *url.URL
}

// parseFlags parses command line flags using the given flag set.
// It returns the processed URL matches, and the HTTP client timeout to use.
func parseFlags(set *flag.FlagSet, args []string) ([]urlMatch, time.Duration, error) {
	matches := make([]urlMatch, 0, len(args)/2)
	set.Func("url", "[<OS>/<architecture>=]URL (at least one required to match)", func(s string) error {
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
			matches = append(matches, urlMatch{pattern, u})
			return nil
		}
	})
	var d time.Duration
	set.DurationVar(&d, "http-timeout", 5*time.Minute, "HTTP client timeout")
	if err := set.Parse(args); err != nil {
		return nil, 0, err
	}
	return matches, d, nil
}

// selectURL selects a URL for a system from the given matches.
func selectURL(s string, urlMatches []urlMatch) (*url.URL, error) {
	for _, um := range urlMatches {
		match, err := filepath.Match(um.pattern, s)
		if err != nil {
			return nil, err
		}
		if match {
			return um.url, nil
		}
	}
	return nil, nil
}

func main() {
	// Basics

	rc := 0
	defer func() {
		os.Exit(rc)
	}()
	prog := filepath.Base(os.Args[0])

	// Set up output

	verboseEnvVar := strings.ToUpper(prog) + "_VERBOSE"
	var verbose *bool
	if s, ok := os.LookupEnv(verboseEnvVar); ok {
		v, _ := strconv.ParseBool(s)
		verbose = &v
	}
	_out := func(w io.Writer, level, format string, a ...any) {
		_, _ = fmt.Fprintf(w, prog+": "+level+": "+format+"\n", a...)
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

	// Process flags

	cacheDirEnvVar := strings.ToUpper(prog) + "_CACHE_HOME"

	var (
		err         error
		urlMatches  []urlMatch
		httpTimeout time.Duration
	)
	flagSet := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	if urlMatches, httpTimeout, err = parseFlags(flagSet, os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Printf(`
%s downloads, caches, and runs executables.
The same one command works for multiple OS/architectures.

The runtime OS and architecture are matched against the given URL matchers.
The first matching one in the order given is chosen as the URL to download.
The matcher OS and architecture may be globs.
As a special case, a plain URL with no matcher part is treated as if it was given with the matcher */*.
URL fragments are treated as hex encoded digests for the download, and checked.

The first non-flag argument or -- terminates %s arguments.
Remaining ones are passed to the downloaded one.

Environment variables:
- %s: location of the cache, defaults to %s in the user cache dir
- %s: controls output verbosity; false decreases, true increases
`, prog, prog, cacheDirEnvVar, prog, verboseEnvVar)
		} else {
			errorOut("parse flags: %v", err)
			rc = 2 // usage
		}
		return
	}

	// Figure out download URL

	osArch := runtime.GOOS + "/" + runtime.GOARCH
	ur, err := selectURL(osArch, urlMatches)
	if err != nil {
		errorOut("select URL: %v", err)
		rc = 2 // usage, bad pattern
		return
	}
	if ur == nil {
		errorOut("no URL available for OS/architecture %s", osArch)
		rc = 1
		return
	}
	infoOut("URL: %s", ur)

	// Set up hashing

	hshType, expectedDigest, err := prepareHash(ur.Fragment)
	if err != nil {
		errorOut("prepare hash: %v", err)
		rc = 1
		return
	}

	// Set up cache

	_, exeBase := path.Split(ur.Path)
	var exePath string
	if cacheDir := os.Getenv(strings.ToUpper(prog) + "_CACHE_HOME"); cacheDir != "" {
		exePath = filepath.Join(cacheDir, exeBase)
		err = os.MkdirAll(cacheDir, 0o777)
	} else {
		exePath, err = xdg.CacheFile(filepath.Join(prog, cacheVersion, urlDir(ur), exeBase))
	}
	if err != nil {
		errorOut("cache setup: %v", err)
		rc = 1
		return
	}
	infoOut("path to executable: %s", exePath)

	// exec from cache

	exec := func(exe string) error {
		args := make([]string, len(flagSet.Args())+1)
		args[0] = exe
		copy(args[1:], flagSet.Args())
		infoOut("exec: %v", args)
		return syscall.Exec(exe, args, os.Environ())
	}

	if err = exec(exePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			infoOut("exec cached: %v", err)
		} else {
			warnOut("exec cached: %v", err)
		}
	}

	// Set up tempfile for download

	tmpf, err := os.CreateTemp(filepath.Dir(exePath), filepath.Base(exePath))
	if err != nil {
		errorOut("set up tempfile: %v", err)
		rc = 1
		return
	}
	defer func() {
		if rmErr := os.Remove(tmpf.Name()); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) {
			warnOut("remove tempfile: %v", rmErr)
		}
	}()
	defer func() {
		_ = tmpf.Close() // ignore error, we may have eagerly closed it already // TODO don't close multiple times, results are undefined
	}()

	// Download

	hc := http.Client{
		Timeout: httpTimeout,
	}
	req, err := http.NewRequest(http.MethodGet, ur.String(), nil)
	if err != nil {
		errorOut("prepare request: %v", err)
		rc = 1
		return
	}
	req.Header.Set("User-Agent", prog+"/"+version)
	// TODO if no checksum, do conditional get: If-None-Match, If-Modified-Since?

	resp, err := hc.Do(req)
	if err != nil {
		errorOut("get %s: %v", ur.String(), err)
		rc = 1
		return
	}
	defer func() {
		_ = resp.Body.Close() // ignore error, we may have eagerly closed it already // TODO don't close multiple times, results are undefined
	}()

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
	if err != nil {
		errorOut("download: %v", err)
		rc = 1
		return
	}
	if err = resp.Body.Close(); err != nil {
		warnOut("close response: %v", err)
	}
	if err = tmpf.Close(); err != nil {
		errorOut("close tempfile: %v", err)
		rc = 1
		return
	}
	infoOut("downloaded %d bytes", n)

	// Check digest

	if hsh != nil {
		gotDigest := hsh.Sum(nil)
		if bytes.Equal(gotDigest, expectedDigest) {
			infoOut("%s digest match: %s", hshType, hex.EncodeToString(expectedDigest))
		} else {
			errorOut("%s digest mismatch: got %s, expected %s", hex.EncodeToString(gotDigest), hex.EncodeToString(expectedDigest))
			rc = 1
			return
		}
	}

	// Move to final location, make executable

	if err = os.Rename(tmpf.Name(), exePath); err != nil {
		errorOut("rename tempfile: %v", err)
		rc = 1
		return
	}
	if err = makeExecutable(exePath); err != nil {
		errorOut("make executable: %v", err)
		rc = 1
		return
	}

	// Write metadata

	data, err := json.Marshal(meta)
	if err != nil {
		warnOut("encode metadata: %v", err)
	} else if err = os.WriteFile(exePath+"-metadata.json", data, 0o666); err != nil {
		warnOut("write metadata: %v", err)
	}

	// Execute

	err = exec(exePath)
	errorOut("exec: %v", err)
	rc = 1
}
