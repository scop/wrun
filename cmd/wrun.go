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
	"bytes"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
)

type Wrun struct {
	ProgName   string
	httpClient *http.Client
	verbose    *bool
}

func NewWrun(progName string) *Wrun {
	w := &Wrun{
		ProgName:   progName,
		httpClient: &http.Client{},
	}
	if s, ok := os.LookupEnv(verboseEnvVar); ok {
		v, _ := strconv.ParseBool(s)
		w.verbose = &v
	}

	return w
}

type level = string

const (
	levelBug   level = "BUG"
	levelError level = "ERR"
	levelWarn  level = "WARN"
	levelInfo  level = "INFO"
)

var levelFormat = fmt.Sprintf("%%%-ds", len(levelInfo))

func (w *Wrun) logFormat(lvl level, format string, args ...any) string {
	s := fmt.Sprintf(w.ProgName+": "+fmt.Sprintf(levelFormat, lvl)+": "+format, args...)

	return s
}

func (w *Wrun) LogInfo(format string, args ...any) {
	if w.verbose != nil && *w.verbose {
		fmt.Fprintln(os.Stderr, w.logFormat(levelInfo, format, args...))
	}
}

func (w *Wrun) LogWarn(format string, args ...any) {
	if w.verbose == nil || *w.verbose {
		fmt.Fprintln(os.Stderr, w.logFormat(levelWarn, format, args...))
	}
}

func (w *Wrun) LogError(format string, args ...any) {
	fmt.Fprintln(os.Stderr, w.logFormat(levelError, format, args...))
}

func (w *Wrun) LogBug(format string, args ...any) {
	panic(w.logFormat(levelBug, format, args...))
}

// HTTPGet sends an HTTP GET request to url with headers.
// It returns the HTTP response and any encountered error.
// An error is also returned on responses having status other than 200.
// headers are colon separated name:value strings.
func (w *Wrun) HTTPGet(url string, headers ...string) (*http.Response, error) {
	const method = http.MethodGet
	w.LogInfo("%s %s", method, url)
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%s %s new request: %w", method, url, err)
	}
	req.Header.Set("User-Agent", w.ProgName+"/"+version)
	for _, h := range headers {
		if k, v, found := strings.Cut(h, ":"); found {
			req.Header.Set(k, v)
		} else {
			return nil, fmt.Errorf("%s %s set request headers: no colon in header: %q", req.Method, url, h)
		}
	}
	// TODO if no checksum, do conditional get: If-None-Match, If-Modified-Since?

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", req.Method, url, err)
	}
	if resp.StatusCode != http.StatusOK {
		if err = resp.Body.Close(); err != nil {
			w.LogWarn("close HTTP response: %v", err)
		}

		return nil, fmt.Errorf("%s %s: HTTP status %s", req.Method, url, resp.Status)
	}

	return resp, nil
}

func (w *Wrun) Download(resp *http.Response, dest io.Writer, hsh hash.Hash, expectedDigest []byte) error {
	var wr io.Writer
	switch {
	case dest == nil && hsh == nil:
		w.LogBug("%s", "destination or hash required to download")
	case dest == nil:
		wr = hsh
	case hsh == nil:
		wr = dest
	default:
		wr = io.MultiWriter(dest, hsh)
	}

	n, err := io.Copy(wr, resp.Body)
	if cErr := resp.Body.Close(); cErr != nil {
		w.LogWarn("close HTTP response: %v", cErr)
	}
	if err != nil {
		return fmt.Errorf("copy stream: %w", err)
	}
	var gotDigest []byte
	if hsh == nil {
		w.LogInfo("got %d bytes", n)
	} else {
		gotDigest = hsh.Sum(nil)
		w.LogInfo("got %d bytes, digest %x", n, gotDigest)
	}

	var digestErr error
	if expectedDigest != nil {
		if bytes.Equal(gotDigest, expectedDigest) {
			w.LogInfo("digest match: %x", gotDigest)
		} else {
			digestErr = fmt.Errorf("digest mismatch: expected %x, got %x", expectedDigest, gotDigest)
		}
	}
	var closeErr error
	if c, ok := dest.(io.Closer); ok {
		if closeErr = c.Close(); closeErr != nil {
			closeErr = fmt.Errorf("close destination: %w", closeErr)
		}
	}

	return errors.Join(digestErr, closeErr)
}

func (w *Wrun) SetUpTempfile(url, dir string) (f *os.File, cleanup func(), err error) {
	// Use temp filename _prefix_, archiver recognizes by filename extension
	tmpName := strings.ToLower(path.Base(url))
	if strings.HasSuffix(tmpName, ".whl") {
		tmpName = strings.TrimSuffix(tmpName, ".whl") + ".zip" // Make archiver recognize it
	}
	f, err = os.CreateTemp(dir, "wrun*-"+tmpName)
	if err != nil {
		return nil, nil, fmt.Errorf("set up tempfile: %w", err)
	}
	cleanup = func() {
		if closeErr := f.Close(); closeErr != nil && !errors.Is(closeErr, os.ErrClosed) {
			w.LogWarn("close tempfile: %v", closeErr)
		}
		if rmErr := os.Remove(f.Name()); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) {
			w.LogWarn("remove tempfile: %v", rmErr)
		}
	}

	return
}
