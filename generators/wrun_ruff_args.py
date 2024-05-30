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

from argparse import ArgumentParser
import hashlib
from urllib.parse import urljoin, quote as urlquote
from urllib.request import urlopen

from wrun_utils import latest_atom_version


file_os_archs = {
    "aarch64-apple-darwin.tar.gz": "darwin/arm64",
    "aarch64-pc-windows-msvc.zip": "windows/arm64",
    # "aarch64-unknown-linux-gnu.tar.gz": None,  # using musl one
    "aarch64-unknown-linux-musl.tar.gz": "linux/arm64",
    # "armv7-unknown-linux-gnueabihf.tar.gz": None,  # using musl one
    "armv7-unknown-linux-musleabihf.tar.gz": "linux/arm",
    "i686-pc-windows-msvc.zip": "windows/386",
    # "i686-unknown-linux-gnu.tar.gz": None,  # using musl one
    "i686-unknown-linux-musl.tar.gz": "linux/386",
    "powerpc64-unknown-linux-gnu.tar.gz": "linux/ppc64",
    "powerpc64le-unknown-linux-gnu.tar.gz": "linux/ppc64le",
    "s390x-unknown-linux-gnu.tar.gz": "linux/s390x",
    "x86_64-apple-darwin.tar.gz": "darwin/amd64",
    "x86_64-pc-windows-msvc.zip": "windows/amd64",
    # "x86_64-unknown-linux-gnu.tar.gz": None,  # using musl one
    "x86_64-unknown-linux-musl.tar.gz": "linux/amd64",
}


def check_hexdigest(expected: str, algo: str, url: str | None) -> None:
    try:
        assert len(expected) == len(hashlib.new(algo, b"canary").hexdigest())
        _ = bytes.fromhex(expected)
    except Exception as e:
        raise ValueError(f'not a {algo} hex digest: "{expected}"') from e
    if not url:
        return
    with urlopen(url) as f:
        got = hashlib.file_digest(f, algo).hexdigest()
    if got != expected:
        raise ValueError(f'{algo} mismatch for "{url}", expected {expected}, got {got}')


def main(version: str, verify: bool) -> None:
    if not version:
        version = latest_atom_version("https://github.com/astral-sh/ruff/releases.atom")

    base_url = (
        f"https://github.com/astral-sh/ruff/releases/download/{urlquote(version)}/"
    )
    version_number = version.lstrip("v")

    for suffix, os_arch in file_os_archs.items():
        fn = f"ruff-{version_number}-{suffix}"
        url = urljoin(base_url, urlquote(fn + ".sha256"))
        with urlopen(url) as f:
            for line in f:
                sline = line.decode()

                hexdigest_filename = sline.split(maxsplit=3)
                if len(hexdigest_filename) != 2:
                    raise ValueError(f'invalid checksums line in {fn}: "{sline}"')
                hexdigest, filename = hexdigest_filename
                filename = filename.lstrip("*")  # at least some windows ones have this
                if filename != fn:
                    raise KeyError(f'unexpected filename in {url}: "{filename}"')

                url = urljoin(base_url, filename)
                check_hexdigest(hexdigest, "sha256", url if verify else None)

                print(f"-url {os_arch}={url}#sha256-{hexdigest}")
    print("-archive-exe-path ruff")


if __name__ == "__main__":
    parser = ArgumentParser()
    parser.add_argument("version", metavar="VERSION", nargs="?")
    parser.add_argument("--skip-verify", dest="verify", action="store_false")
    args = parser.parse_args()
    main(args.version, args.verify)
