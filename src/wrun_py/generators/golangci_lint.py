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
golangci_lint.py -- generate wrun command line args for golangci-lint

* https://golangci-lint.run
* https://github.com/golangci/golangci-lint/releases
"""

from argparse import ArgumentParser
import re
from urllib.parse import urljoin, quote as urlquote
from urllib.request import urlopen

from . import check_hexdigest, latest_atom_version


def main() -> None:
    parser = ArgumentParser()
    parser.add_argument("version", metavar="VERSION", nargs="?")
    parser.add_argument("--skip-verify", dest="verify", action="store_false")
    args = parser.parse_args()

    if not args.version:
        args.version = latest_atom_version(
            "https://github.com/golangci/golangci-lint/releases.atom"
        )

    version_number = args.version.lstrip("v")
    base_url = f"https://github.com/golangci/golangci-lint/releases/download/{urlquote(args.version)}/"
    archive_exe_paths = []

    with urlopen(
        urljoin(base_url, urlquote(f"golangci-lint-{version_number}-checksums.txt"))
    ) as f:
        for line in f:
            sline = line.decode()

            hexdigest_filename = sline.split(maxsplit=2)
            if len(hexdigest_filename) != 2:
                raise ValueError(f'invalid checksums line: "{sline}"')
            hexdigest, filename = hexdigest_filename

            if m := re.search(
                r"^(golangci-lint-(.+)-([^-]+)-([^-]+))\.(?:t[\w.]+|zip)$", filename
            ):
                if m.group(2) != version_number:
                    continue
                dirname = m.group(1)
                os = m.group(3)
                arch = m.group(4)
            else:
                continue
            if arch == "armv7":
                continue  # using armv6 one
            if arch == "armv6":
                arch = "arm"

            url = urljoin(base_url, filename)
            check_hexdigest(hexdigest, "sha256", url if args.verify else None)

            print(f"-url {os}/{arch}={url}#sha256-{hexdigest}")
            suffix = ".exe" if os == "windows" else ""
            archive_exe_paths.append(
                f"-archive-exe-path {os}/{arch}={dirname}/golangci-lint{suffix}"
            )
    for p in archive_exe_paths:
        print(p)


if __name__ == "__main__":
    main()
