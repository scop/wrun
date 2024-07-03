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
	"crypto"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"strconv"
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

var levelFormat = fmt.Sprintf("%%%ds", len(levelInfo))

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

func (w *Wrun) HTTPGet(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("prepare %s %s request: %w", http.MethodGet, url, err)
	}
	req.Header.Set("User-Agent", w.ProgName+"/"+version)
	// TODO if no checksum, do conditional get: If-None-Match, If-Modified-Since?

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", req.Method, url, err)
	}
	return resp, nil
}

func (w *Wrun) Download(resp *http.Response, dest io.Writer, hshType crypto.Hash, hsh hash.Hash, expectedDigest []byte) error {
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
	w.LogInfo("downloaded %d bytes", n)

	var digestErr error
	if expectedDigest != nil {
		gotDigest := hsh.Sum(nil)
		expectedHex := hex.EncodeToString(expectedDigest)
		if bytes.Equal(gotDigest, expectedDigest) {
			w.LogInfo("%s digest match: %s", hshType, expectedHex)
		} else {
			digestErr = fmt.Errorf("%s digest mismatch: expected %s, got %s", hshType, expectedHex, hex.EncodeToString(gotDigest))
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
