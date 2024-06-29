#!/usr/bin/env python3

# Copyright 2024 Ville Skyttä
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
hadolint.py -- generate wrun command line args for hadolint

* https://hadolint.github.io/hadolint/
* https://github.com/hadolint/hadolint/releases
"""

from argparse import ArgumentParser
from urllib.parse import urljoin, quote as urlquote
from urllib.request import urlopen

from . import check_hexdigest, latest_atom_version


file_os_archs = {
    "Darwin-x86_64": "darwin/amd64",
    "Linux-arm64": "linux/arm64",
    "Linux-x86_64": "linux/amd64",
    "Windows-x86_64.exe": "windows/amd64",
}


def main() -> None:
    parser = ArgumentParser()
    parser.add_argument("version", metavar="VERSION", nargs="?")
    parser.add_argument("--skip-verify", dest="verify", action="store_false")
    args = parser.parse_args()

    if not args.version:
        args.version = latest_atom_version(
            "https://github.com/hadolint/hadolint/releases.atom"
        )

    base_url = f"https://github.com/hadolint/hadolint/releases/download/{urlquote(args.version)}/"

    for suffix, os_arch in file_os_archs.items():
        fn = f"hadolint-{suffix}"
        url = urljoin(base_url, urlquote(fn + ".sha256"))
        with urlopen(url) as f:
            for line in f:
                sline = line.decode()

                hexdigest_filename = sline.split(maxsplit=2)
                if len(hexdigest_filename) != 2:
                    raise ValueError(f'invalid checksums line in {fn}: "{sline}"')
                hexdigest, filename = hexdigest_filename
                filename = filename.lstrip("*")
                if filename != fn:
                    raise KeyError(f'unexpected filename in {url}: "{filename}"')

                url = urljoin(base_url, filename)
                check_hexdigest(hexdigest, "sha256", url if args.verify else None)

                print(f"--url {os_arch}={url}#sha256-{hexdigest}")


if __name__ == "__main__":
    main()
