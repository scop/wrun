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
black.py -- generate wrun command line args for black

* https://black.readthedocs.io
* https://github.com/psf/black/releases
"""

from argparse import ArgumentParser
from urllib.parse import urljoin, quote as urlquote
from urllib.request import urlopen

from . import file_digest, latest_atom_version


file_os_archs = {
    "black_linux": "linux/amd64",
    "black_macos": "darwin/amd64",
    "black_windows.exe": "windows/amd64",
}


def main() -> None:
    parser = ArgumentParser()
    parser.add_argument("version", metavar="VERSION", nargs="?")
    args = parser.parse_args()

    if not args.version:
        args.version = latest_atom_version("https://github.com/psf/black/releases.atom")

    base_url = (
        f"https://github.com/psf/black/releases/download/{urlquote(args.version)}/"
    )

    for filename, os_arch in file_os_archs.items():
        url = urljoin(base_url, filename)
        with urlopen(url) as f:
            digest = file_digest(f, "sha256")

        print(f"-url {os_arch}={url}#sha256-{digest.hexdigest()}")


if __name__ == "__main__":
    main()
