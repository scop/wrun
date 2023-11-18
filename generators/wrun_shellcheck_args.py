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
wrun_shellcheck_args.py -- generate wrun command line args for shellcheck

* https://www.shellcheck.net
* https://github.com/koalaman/shellcheck/releases
"""

from argparse import ArgumentParser
import hashlib
from urllib.parse import urljoin, quote as urlquote
from urllib.request import urlopen

file_os_archs = {
    "shellcheck-{version}.darwin.x86_64.tar.xz": "darwin/amd64",
    "shellcheck-{version}.linux.aarch64.tar.xz": "linux/arm64",
    "shellcheck-{version}.linux.armv6hf.tar.xz": "linux/arm",
    "shellcheck-{version}.linux.x86_64.tar.xz": "linux/amd64",
    "shellcheck-{version}.zip": "windows/amd64",
}


def main(version: str) -> None:
    base_url = (
        f"https://github.com/koalaman/shellcheck/releases/download/{urlquote(version)}/"
    )

    for fn, os_arch in file_os_archs.items():
        url = urljoin(base_url, urlquote(fn.format(version=version)))
        with urlopen(url) as f:
            digest = hashlib.file_digest(f, "sha256")

        print(f"-url {os_arch}={url}#sha256-{digest.hexdigest()}")
    print("-archive-exe-path windows/*=shellcheck.exe")
    print(f"-archive-exe-path shellcheck-{version}/shellcheck")


if __name__ == "__main__":
    parser = ArgumentParser()
    parser.add_argument("version", metavar="VERSION")
    args = parser.parse_args()
    main(args.version)
