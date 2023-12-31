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
wrun_tflint_args.py -- generate wrun command line args for tflint

* https://github.com/terraform-linters/tflint
* https://github.com/terraform-linters/tflint/releases
"""

from argparse import ArgumentParser
import hashlib
from urllib.parse import urljoin, quote as urlquote
from urllib.request import urlopen

file_os_archs = {
    "tflint_darwin_amd64.zip": "darwin/amd64",
    "tflint_darwin_arm64.zip": "darwin/arm64",
    "tflint_linux_386.zip": "linux/386",
    "tflint_linux_amd64.zip": "linux/amd64",
    "tflint_linux_arm.zip": "linux/arm",
    "tflint_linux_arm64.zip": "linux/arm64",
    "tflint_windows_386.zip": "windows/386",
    "tflint_windows_amd64.zip": "windows/amd64",
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
    base_url = f"https://github.com/terraform-linters/tflint/releases/download/{urlquote(version)}/"

    with urlopen(urljoin(base_url, "checksums.txt")) as f:
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
            check_hexdigest(hexdigest, "sha256", url if verify else None)

            print(f"-url {os_arch}={url}#sha256-{hexdigest}")
    print("-archive-exe-path tflint")


if __name__ == "__main__":
    parser = ArgumentParser()
    parser.add_argument("version", metavar="VERSION")
    parser.add_argument("--skip-verify", dest="verify", action="store_false")
    args = parser.parse_args()
    main(args.version, args.verify)
