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
committed.py -- generate wrun command line args for committed

* https://github.com/crate-ci/committed
* https://pypi.org/project/committed/#files
* https://warehouse.pypa.io/api-reference/json.html
"""

from argparse import ArgumentParser
from fnmatch import fnmatch
import json
from urllib.parse import quote as urlquote
from urllib.request import urlopen

from . import check_hexdigest, latest_rss2_version


file_os_archs = {
    "committed-VERSION_NUMBER-py3-none-macosx_*_x86_64.whl": "darwin/amd64",
    "committed-VERSION_NUMBER-py3-none-macosx_*_arm64.whl": "darwin/arm64",
    "committed-VERSION_NUMBER-py3-none-manylinux_*_aarch64.manylinux*_aarch64.whl": None,  # using musl one
    "committed-VERSION_NUMBER-py3-none-manylinux_*_i686.manylinux*_i686.whl": "linux/386",
    "committed-VERSION_NUMBER-py3-none-manylinux_*_x86_64.manylinux*_x86_64.whl": None,  # using musl one
    "committed-VERSION_NUMBER-py3-none-musllinux_*_aarch64.whl": "linux/arm64",
    "committed-VERSION_NUMBER-py3-none-musllinux_*_x86_64.whl": "linux/amd64",
    "committed-VERSION_NUMBER-py3-none-win32.whl": "windows/386",
    "committed-VERSION_NUMBER-py3-none-win_amd64.whl": "windows/amd64",
}


def main() -> None:
    parser = ArgumentParser()
    parser.add_argument("version", metavar="VERSION", nargs="?")
    parser.add_argument("--skip-verify", dest="verify", action="store_false")
    args = parser.parse_args()

    project = "committed"
    if not args.version:
        args.version = latest_rss2_version(
            f"https://pypi.org/rss/project/{urlquote(project)}/releases.xml"
        )

    version_number = args.version.lstrip("v")
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

        lookup_filename = filename.replace(f"-{version_number}-", "-VERSION_NUMBER-", 1)
        os_arch_found = False
        for file_glob, file_os_arch in file_os_archs.items():
            if fnmatch(lookup_filename, file_glob):
                if os_arch_found:
                    raise KeyError(f"multiple os/arch matches for {filename}")
                os_arch_found = True
                os_arch = file_os_arch
        if not os_arch_found:
            raise KeyError(f"unhandled file: {filename}")
        if os_arch is None:
            continue

        check_hexdigest(hexdigest, "sha256", url["url"] if args.verify else None)

        print(f"--url {os_arch}={url['url']}#sha256-{hexdigest}")
        suffix = ".exe" if os_arch.startswith("windows/") else ""
        archive_exe_paths.append(
            f"--archive-exe-path {os_arch}={project}-{version_number}.data/scripts/{project}{suffix}"
        )
    for p in archive_exe_paths:
        print(p)


if __name__ == "__main__":
    main()
