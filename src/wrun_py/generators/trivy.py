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
trivy.py -- generate wrun command line args for trivy

* https://trivy.dev
* https://github.com/aquasecurity/trivy/releases
"""

from argparse import ArgumentParser
from urllib.parse import urljoin, quote as urlquote
from urllib.request import urlopen

from . import check_hexdigest, latest_atom_version


file_os_archs = {
    "trivy_{version_number}_FreeBSD-32bit.tar.gz": "freebsd/386",
    "trivy_{version_number}_FreeBSD-64bit.tar.gz": "freebsd/amd64",
    "trivy_{version_number}_Linux-32bit.tar.gz": "linux/386",
    "trivy_{version_number}_Linux-64bit.tar.gz": "linux/amd64",
    "trivy_{version_number}_Linux-ARM.tar.gz": "linux/arm",
    "trivy_{version_number}_Linux-ARM64.tar.gz": "linux/arm64",
    "trivy_{version_number}_Linux-PPC64LE.tar.gz": "linux/ppc64le",
    "trivy_{version_number}_Linux-s390x.tar.gz": "linux/s390x",
    "trivy_{version_number}_macOS-64bit.tar.gz": "darwin/amd64",
    "trivy_{version_number}_macOS-ARM64.tar.gz": "darwin/arm64",
    "trivy_{version_number}_windows-64bit.zip": "windows/amd64",
}


def main() -> None:
    parser = ArgumentParser()
    parser.add_argument("version", metavar="VERSION", nargs="?")
    parser.add_argument("--skip-verify", dest="verify", action="store_false")
    args = parser.parse_args()

    if not args.version:
        args.version = latest_atom_version(
            "https://github.com/aquasecurity/trivy/releases.atom"
        )

    version_number = args.version.lstrip("v")
    base_url = f"https://github.com/aquasecurity/trivy/releases/download/{urlquote(args.version)}/"

    with urlopen(urljoin(base_url, f"trivy_{version_number}_checksums.txt")) as f:
        for line in f:
            sline = line.decode()

            hexdigest_filename = sline.split(maxsplit=2)
            if len(hexdigest_filename) != 2:
                raise ValueError(f'invalid checksums line: "{sline}"')
            hexdigest, filename = hexdigest_filename

            lookup_filename = filename.replace(
                f"_{version_number}_", "_{version_number}_", 1
            )
            if lookup_filename not in file_os_archs:
                if lookup_filename.endswith(".deb") or lookup_filename.endswith(".rpm"):
                    continue
                raise KeyError(f'unhandled file: "{filename}"')
            os_arch = file_os_archs[lookup_filename]
            if os_arch is None:
                continue

            url = urljoin(base_url, filename)
            check_hexdigest(hexdigest, "sha256", url if args.verify else None)

            print(f"--url {os_arch}={url}#sha256-{hexdigest}")
    print("--archive-exe-path trivy")


if __name__ == "__main__":
    main()
