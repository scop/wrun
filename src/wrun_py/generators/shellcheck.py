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
shellcheck.py -- generate wrun command line args for shellcheck

* https://www.shellcheck.net
* https://github.com/koalaman/shellcheck/releases
"""

from argparse import ArgumentParser
from urllib.parse import urljoin, quote as urlquote
from urllib.request import urlopen

from . import file_digest, latest_atom_version


file_os_archs = {
    "shellcheck-{version}.darwin.aarch64.tar.xz": "darwin/arm64",
    "shellcheck-{version}.darwin.x86_64.tar.xz": "darwin/amd64",
    "shellcheck-{version}.linux.aarch64.tar.xz": "linux/arm64",
    "shellcheck-{version}.linux.armv6hf.tar.xz": "linux/arm",
    "shellcheck-{version}.linux.x86_64.tar.xz": "linux/amd64",
    "shellcheck-{version}.zip": "windows/amd64",
}


def main() -> None:
    parser = ArgumentParser()
    parser.add_argument("version", metavar="VERSION", nargs="?")
    args = parser.parse_args()

    if not args.version:
        args.version = latest_atom_version(
            "https://github.com/koalaman/shellcheck/releases.atom"
        )

    base_url = f"https://github.com/koalaman/shellcheck/releases/download/{urlquote(args.version)}/"

    for fn, os_arch in file_os_archs.items():
        url = urljoin(base_url, urlquote(fn.format(version=args.version)))
        with urlopen(url) as f:
            digest = file_digest(f, "sha256")

        print(f"-url {os_arch}={url}#sha256-{digest.hexdigest()}")
    print("-archive-exe-path windows/*=shellcheck.exe")
    print(f"-archive-exe-path shellcheck-{args.version}/shellcheck")


if __name__ == "__main__":
    main()
