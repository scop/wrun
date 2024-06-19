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
dprint.py -- generate wrun command line args for dprint

* https://dprint.dev
* https://github.com/dprint/dprint/releases
"""

from argparse import ArgumentParser
from urllib.parse import urljoin, quote as urlquote
from urllib.request import urlopen

from . import check_hexdigest, latest_atom_version


file_os_archs = {
    "dprint-aarch64-apple-darwin.zip": "darwin/arm64",
    "dprint-aarch64-unknown-linux-gnu.zip": None,  # using musl one
    "dprint-aarch64-unknown-linux-musl.zip": "linux/arm64",
    "dprint-x86_64-apple-darwin.zip": "darwin/amd64",
    "dprint-x86_64-pc-windows-msvc-installer.exe": None,
    "dprint-x86_64-pc-windows-msvc.zip": "windows/amd64",
    "dprint-x86_64-unknown-linux-gnu.zip": None,  # using musl one
    "dprint-x86_64-unknown-linux-musl.zip": "linux/amd64",
}


def main() -> None:
    parser = ArgumentParser()
    parser.add_argument("version", metavar="VERSION", nargs="?")
    parser.add_argument("--skip-verify", dest="verify", action="store_false")
    args = parser.parse_args()

    if not args.version:
        args.version = latest_atom_version(
            "https://github.com/dprint/dprint/releases.atom"
        )

    base_url = (
        f"https://github.com/dprint/dprint/releases/download/{urlquote(args.version)}/"
    )

    with urlopen(urljoin(base_url, "SHASUMS256.txt")) as f:
        for line in f:
            sline = line.decode()

            hexdigest_filename = sline.split(maxsplit=3)
            if len(hexdigest_filename) != 2:
                raise ValueError(f'invalid checksums line: "{sline}"')
            hexdigest, filename = hexdigest_filename

            if filename not in file_os_archs:
                raise KeyError(f'unhandled file: "{filename}"')
            os_arch = file_os_archs[filename]
            if os_arch is None:
                continue

            url = urljoin(base_url, filename)
            check_hexdigest(hexdigest, "sha256", url if args.verify else None)

            print(f"-url {os_arch}={url}#sha256-{hexdigest}")
    print("-archive-exe-path dprint")


if __name__ == "__main__":
    main()
