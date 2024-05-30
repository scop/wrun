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
wrun_typos_args.py -- generate wrun command line args for typos

* https://github.com/crate-ci/typos
* https://pypi.org/project/typos/#files
* https://warehouse.pypa.io/api-reference/json.html
"""

from argparse import ArgumentParser
import hashlib
import json
from urllib.parse import quote as urlquote
from urllib.request import urlopen

from wrun_utils import latest_rss2_version


file_os_archs = {
    "typos-{version_number}-py3-none-macosx_10_7_x86_64.whl": "darwin/amd64",
    "typos-{version_number}-py3-none-macosx_11_0_arm64.whl": "darwin/arm64",
    "typos-{version_number}-py3-none-manylinux_2_17_aarch64.manylinux2014_aarch64.whl": None,  # using musl one
    "typos-{version_number}-py3-none-manylinux_2_17_i686.manylinux2014_i686.whl": "linux/386",
    "typos-{version_number}-py3-none-manylinux_2_17_x86_64.manylinux2014_x86_64.whl": None,  # using musl one
    "typos-{version_number}-py3-none-musllinux_1_2_aarch64.whl": "linux/arm64",
    "typos-{version_number}-py3-none-musllinux_1_2_x86_64.whl": "linux/amd64",
    "typos-{version_number}-py3-none-win32.whl": "windows/386",
    "typos-{version_number}-py3-none-win_amd64.whl": "windows/amd64",
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
    if not version:
        version = latest_rss2_version("https://pypi.org/rss/project/typos/releases.xml")

    project = "typos"
    version_number = version.lstrip("v")
    archive_exe_paths = []

    release_url = (
        f"https://pypi.org/pypi/{urlquote(project)}/{urlquote(version_number)}/json"
    )
    with urlopen(release_url) as f:
        release_data = json.load(f)

    for url in release_data["urls"]:
        if url["packagetype"] != "bdist_wheel":
            continue

        filename = url["filename"]

        try:
            hexdigest = url["digests"]["sha256"]
        except KeyError as e:
            raise KeyError(f"no sha256 digest available for {filename}") from e

        lookup_filename = filename.replace(
            f"-{version_number}-", "-{version_number}-", 1
        )
        if lookup_filename not in file_os_archs:
            raise KeyError(f'unhandled file: "{filename}"')
        os_arch = file_os_archs[lookup_filename]
        if os_arch is None:
            continue

        check_hexdigest(hexdigest, "sha256", url["url"] if verify else None)

        print(f"-url {os_arch}={url['url']}#sha256-{hexdigest}")
        suffix = ".exe" if os_arch.startswith("windows/") else ""
        archive_exe_paths.append(
            f"-archive-exe-path {os_arch}={project}-{version_number}.data/scripts/{project}{suffix}"
        )
    for p in archive_exe_paths:
        print(p)


if __name__ == "__main__":
    parser = ArgumentParser()
    parser.add_argument("version", metavar="VERSION", nargs="?")
    parser.add_argument("--skip-verify", dest="verify", action="store_false")
    args = parser.parse_args()
    main(args.version, args.verify)
