#!/usr/bin/env python3

# Copyright 2023 Ville Skyttä
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# SPDX-License-Identifier: Apache-2.0


"""
wrun_ruff_args.py -- generate wrun command line args for ruff

* https://docs.astral.sh/ruff/
* https://github.com/astral-sh/ruff/releases
"""

import hashlib
import sys
from urllib.parse import urljoin, quote as urlquote
from urllib.request import urlopen

file_os_archs = {
    "ruff-aarch64-apple-darwin.tar.gz": "darwin/arm64",
    "ruff-aarch64-pc-windows-msvc.zip": "windows/arm64",
    "ruff-aarch64-unknown-linux-gnu.tar.gz": None,  # using musl one
    "ruff-aarch64-unknown-linux-musl.tar.gz": "linux/arm64",
    "ruff-armv7-unknown-linux-gnueabihf.tar.gz": None,  # using musl one
    "ruff-armv7-unknown-linux-musleabihf.tar.gz": "linux/arm",
    "ruff-i686-pc-windows-msvc.zip": "windows/386",
    "ruff-i686-unknown-linux-gnu.tar.gz": None,  # using musl one
    "ruff-i686-unknown-linux-musl.tar.gz": "linux/386",
    "ruff-powerpc64-unknown-linux-gnu.tar.gz": "linux/ppc64",
    "ruff-powerpc64le-unknown-linux-gnu.tar.gz": "linux/ppc64le",
    "ruff-s390x-unknown-linux-gnu.tar.gz": "linux/s390x",
    "ruff-x86_64-apple-darwin.tar.gz": "darwin/amd64",
    "ruff-x86_64-pc-windows-msvc.zip": "windows/amd64",
    "ruff-x86_64-unknown-linux-gnu.tar.gz": None,  # using musl one
    "ruff-x86_64-unknown-linux-musl.tar.gz": "linux/amd64",
}


def check_hexdigest(url: str, algo: str, expected: str) -> None:
    try:
        assert len(expected) == len(hashlib.new(algo, b"canary").hexdigest())
        _ = bytes.fromhex(expected)
    except Exception as e:
        raise ValueError(f'not a {algo} hex digest: "{expected}"') from e
    with urlopen(url) as f:
        got = hashlib.file_digest(f, algo).hexdigest()
    if got != expected:
        raise ValueError(f'{algo} mismatch for "{url}", expected {expected}, got {got}')


def main(version: str) -> None:
    base_url = (
        f"https://github.com/astral-sh/ruff/releases/download/{urlquote(version)}/"
    )

    for fn, os_arch in file_os_archs.items():
        fn += ".sha256"
        url = urljoin(base_url, urlquote(fn))
        with urlopen(url) as f:
            for line in f:
                sline = line.decode()

                hexdigest_filename = sline.split(maxsplit=3)
                if len(hexdigest_filename) != 2:
                    raise ValueError(f'invalid checksums line in {fn}: "{sline}"')
                hexdigest, filename = hexdigest_filename

                filename = filename.lstrip("*")  # Kind of dangerous...
                if filename not in file_os_archs:
                    raise KeyError(f'unhandled file: "{filename}"')
                os_arch = file_os_archs[filename]
                if os_arch is None:
                    continue

                url = urljoin(base_url, filename)
                check_hexdigest(url, "sha256", hexdigest)

                print(f"-url {os_arch}={url}#sha256-{hexdigest}")
    print("-archive-exe-path windows/*=ruff.exe")
    print("-archive-exe-path ruff")


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print(f"usage: {sys.argv[0]} VERSION")
        sys.exit(2)
    main(sys.argv[1])
