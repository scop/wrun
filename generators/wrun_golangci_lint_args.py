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
wrun_golangci_lint_args.py -- generate wrun command line args for golangci-lint

* https://golangci-lint.run
* https://github.com/golangci/golangci-lint/releases
"""

from argparse import ArgumentParser
import hashlib
import re
from urllib.parse import urljoin, quote as urlquote
from urllib.request import urlopen

file_os_archs = {
    "golangci-lint-{version_number}-darwin-amd64.tar.gz": "darwin/amd64",
    "golangci-lint-{version_number}-darwin-arm64.tar.gz": "darwin/arm64",
    "golangci-lint-{version_number}-freebsd-386.tar.gz": "freebsd/386",
    "golangci-lint-{version_number}-freebsd-amd64.tar.gz": "freebsd/amd64",
    "golangci-lint-{version_number}-freebsd-armv6.tar.gz": "freebsd/arm",
    "golangci-lint-{version_number}-freebsd-armv7.tar.gz": None,  # using armv6 one
    "golangci-lint-{version_number}-illumos-amd64.tar.gz": "illumos/amd64",
    "golangci-lint-{version_number}-linux-386.tar.gz": "linux/386",
    "golangci-lint-{version_number}-linux-amd64.tar.gz": "linux/amd64",
    "golangci-lint-{version_number}-linux-arm64.tar.gz": "linux/arm64",
    "golangci-lint-{version_number}-linux-armv6.tar.gz": "linux/arm",
    "golangci-lint-{version_number}-linux-armv7.tar.gz": None,  # using armv6 one
    "golangci-lint-{version_number}-linux-loong64.tar.gz": "linux/loong64",
    "golangci-lint-{version_number}-linux-mips64.tar.gz": "linux/mips64",
    "golangci-lint-{version_number}-linux-mips64le.tar.gz": "linux/mips64le",
    "golangci-lint-{version_number}-linux-ppc64le.tar.gz": "linux/ppc64le",
    "golangci-lint-{version_number}-linux-riscv64.tar.gz": "linux/riscv64",
    "golangci-lint-{version_number}-linux-s390x.tar.gz": "linux/s390x",
    "golangci-lint-{version_number}-netbsd-386.tar.gz": "netbsd/386",
    "golangci-lint-{version_number}-netbsd-amd64.tar.gz": "netbsd/amd64",
    "golangci-lint-{version_number}-netbsd-armv6.tar.gz": "netbsd/arm",
    "golangci-lint-{version_number}-netbsd-armv7.tar.gz": None,  # using armv6 one
    "golangci-lint-{version_number}-source.tar.gz": None,
    "golangci-lint-{version_number}-windows-386.zip": "windows/386",
    "golangci-lint-{version_number}-windows-amd64.zip": "windows/amd64",
    "golangci-lint-{version_number}-windows-arm64.zip": "windows/arm64",
    "golangci-lint-{version_number}-windows-armv6.zip": "windows/arm",
    "golangci-lint-{version_number}-windows-armv7.zip": None,  # using armv6 one
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
    version_number = version.lstrip("v")
    base_url = f"https://github.com/golangci/golangci-lint/releases/download/{urlquote(version)}/"
    archive_exe_paths = []

    with urlopen(
        urljoin(base_url, urlquote(f"golangci-lint-{version_number}-checksums.txt"))
    ) as f:
        for line in f:
            sline = line.decode()

            hexdigest_filename = sline.split(maxsplit=3)
            if len(hexdigest_filename) != 2:
                raise ValueError(f'invalid checksums line: "{sline}"')
            hexdigest, filename = hexdigest_filename

            if filename.endswith(".deb") or filename.endswith(".rpm"):
                continue

            lookup_filename = filename.replace(
                f"-{version_number}-", "-{version_number}-", 1
            )
            if lookup_filename not in file_os_archs:
                raise KeyError(f'unhandled file: "{filename}"')
            os_arch = file_os_archs[lookup_filename]
            if os_arch is None:
                continue

            url = urljoin(base_url, filename)
            check_hexdigest(hexdigest, "sha256", url if verify else None)

            print(f"-url {os_arch}={url}#sha256-{hexdigest}")
            dirname = re.sub(r"\.(t[\w.]+|zip)$", "", filename)
            suffix = ".exe" if os_arch.startswith("windows/") else ""
            archive_exe_paths.append(
                f"-archive-exe-path {os_arch}={dirname}/golangci-lint{suffix}"
            )
    for p in archive_exe_paths:
        print(p)


if __name__ == "__main__":
    parser = ArgumentParser()
    parser.add_argument("version", metavar="VERSION")
    parser.add_argument("--skip-verify", dest="verify", action="store_false")
    args = parser.parse_args()
    main(args.version, args.verify)
