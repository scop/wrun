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
terraform.py -- generate wrun command line args for terraform

* https://www.terraform.io
* https://developer.hashicorp.com/terraform/install
"""

from argparse import ArgumentParser
from urllib.parse import urljoin, quote as urlquote
from urllib.request import urlopen

from . import check_hexdigest, latest_atom_version


file_os_archs = {
    "terraform_{version}_darwin_amd64.zip": "darwin/amd64",
    "terraform_{version}_darwin_arm64.zip": "darwin/arm64",
    "terraform_{version}_freebsd_386.zip": "freebsd/386",
    "terraform_{version}_freebsd_amd64.zip": "freebsd/amd64",
    "terraform_{version}_freebsd_arm.zip": "freebsd/arm",
    "terraform_{version}_linux_386.zip": "linux/386",
    "terraform_{version}_linux_amd64.zip": "linux/amd64",
    "terraform_{version}_linux_arm.zip": "linux/arm",
    "terraform_{version}_linux_arm64.zip": "linux/arm64",
    "terraform_{version}_openbsd_386.zip": "openbsd/386",
    "terraform_{version}_openbsd_amd64.zip": "openbsd/amd64",
    "terraform_{version}_solaris_amd64.zip": "solaris/amd64",
    "terraform_{version}_windows_386.zip": "windows/386",
    "terraform_{version}_windows_amd64.zip": "windows/amd64",
}


def main() -> None:
    parser = ArgumentParser()
    parser.add_argument("version", metavar="VERSION", nargs="?")
    parser.add_argument("--skip-verify", dest="verify", action="store_false")
    args = parser.parse_args()

    if not args.version:
        args.version = latest_atom_version(
            "https://github.com/hashicorp/terraform/releases.atom"
        )
    version_number = args.version.lstrip("v")

    base_url = f"https://releases.hashicorp.com/terraform/{urlquote(version_number)}/"

    with urlopen(
        urljoin(base_url, urlquote(f"terraform_{version_number}_SHA256SUMS"))
    ) as f:
        for line in f:
            sline = line.decode()

            hexdigest_filename = sline.split(maxsplit=3)
            if len(hexdigest_filename) != 2:
                raise ValueError(f'invalid checksums line: "{sline}"')
            hexdigest, filename = hexdigest_filename

            lookup_filename = filename.replace(
                f"_{version_number}_", "_{version_number}_", 1
            )
            if lookup_filename not in file_os_archs:
                raise KeyError(f'unhandled file: "{filename}"')
            os_arch = file_os_archs[lookup_filename]
            if os_arch is None:
                continue

            url = urljoin(base_url, filename)
            check_hexdigest(hexdigest, "sha256", url if args.verify else None)

            print(f"-url {os_arch}={url}#sha256-{hexdigest}")
    print("-archive-exe-path terraform")


if __name__ == "__main__":
    main()
