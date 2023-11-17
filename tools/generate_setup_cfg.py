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

import re
import string
import sys
from urllib.parse import quote as urlquote
from urllib.request import urlopen

# TODO: verify linux/386 (i386), linux/armv6hf, and windows/386 markers
setup_cfg_template = """
[metadata]
name = wrun_py
version = ${python_pkg_version}
description = Web executable launcher
long_description = file: README.md
long_description_content_type = text/markdown
url = https://github.com/scop/wrun
author = Ville Skyttä
author_email = ville.skytta@iki.fi
license = Apache License 2.0
license_files = LICENSE
classifiers =
    Development Status :: 4 - Beta
    Intended Audience :: Developers
    License :: OSI Approved :: Apache Software License
    Operating System :: MacOS
    Operating System :: Microsoft :: Windows
    Operating System :: POSIX :: Linux
    Programming Language :: Go
    Topic :: Internet :: WWW/HTTP
    Topic :: Utilities

[options]
packages =
python_requires = >=3.8
setup_requires =
    setuptools-download

[setuptools_download]
download_scripts =
    [wrun]
    group = wrun-binary
    marker = sys_platform == "darwin" and platform_machine == "x86_64"
    url = ${url_darwin_amd64}
    sha256 = ${sha256_darwin_amd64}
    [wrun]
    group = wrun-binary
    marker = sys_platform == "darwin" and platform_machine == "arm64"
    url = ${url_darwin_arm64}
    sha256 = ${sha256_darwin_arm64}
    [wrun]
    group = wrun-binary
    marker = sys_platform == "linux" and platform_machine == "i386"
    marker = sys_platform == "linux" and platform_machine == "i686"
    url = ${url_linux_386}
    sha256 = ${sha256_linux_386}
    [wrun]
    group = wrun-binary
    marker = sys_platform == "linux" and platform_machine == "x86_64"
    url = ${url_linux_amd64}
    sha256 = ${sha256_linux_amd64}
    [wrun]
    group = wrun-binary
    marker = sys_platform == "linux" and platform_machine == "aarch64"
    url = ${url_linux_arm64}
    sha256 = ${sha256_linux_arm64}
    [wrun]
    group = wrun-binary
    marker = sys_platform == "linux" and platform_machine == "armv6hf"
    marker = sys_platform == "linux" and platform_machine == "armv7l"
    url = ${url_linux_armv6}
    sha256 = ${sha256_linux_armv6}
    [wrun.exe]
    group = wrun-binary
    marker = sys_platform == "win32" and platform_machine == "x86"
    marker = sys_platform == "cygwin" and platform_machine == "i386"
    url = ${url_windows_386}
    sha256 = ${sha256_windows_386}
    [wrun.exe]
    group = wrun-binary
    marker = sys_platform == "win32" and platform_machine == "AMD64"
    marker = sys_platform == "cygwin" and platform_machine == "x86_64"
    url = ${url_windows_amd64}
    sha256 = ${sha256_windows_amd64}
"""


def process_checksums(url: str) -> dict[str, str]:
    baseurl, sep, filename = url.rpartition("/")
    if (baseurl == "" and sep == "") or not filename.endswith(".txt"):
        raise ValueError(f'invalid checksums url: "{url}"')

    parts = str.split(filename, "_")
    if len(parts) < 3:
        raise ValueError(f'invalid checksums filename: "{filename}"')

    data: dict[str, str] = {}

    with urlopen(url) as f:
        for line in f:
            sline = line.decode()
            sha256_filename = sline.split(maxsplit=3)
            if len(sha256_filename) != 2:
                raise ValueError(f'invalid checksums line: "{sline}"')
            sha256, filename = sha256_filename

            if m := re.search(r"_([a-z0-9]+_[a-z0-9]+)(?:\.exe)?$", filename):
                os_arch = m.group(1)
            else:
                continue

            if len(sha256) != 64:
                raise ValueError(f'invalid checksums sha256: "{sha256}"')
            _ = bytes.fromhex(sha256)  # test parse

            data[f"url_{os_arch}"] = f"{baseurl}{sep}{urlquote(filename)}"
            data[f"sha256_{os_arch}"] = sha256

    return data


def main(python_pkg_tag: str) -> None:
    main_tag, _, _ = python_pkg_tag.partition("-")
    main_version = main_tag.lstrip("v")
    checksums_txt_url = f"https://github.com/scop/wrun/releases/download/{urlquote(main_tag)}/wrun_{urlquote(main_version)}_checksums.txt"
    data = process_checksums(checksums_txt_url)
    data["python_pkg_version"] = python_pkg_tag.lstrip("v")
    st = string.Template(setup_cfg_template.lstrip())
    setup_cfg = st.substitute(data)
    sys.stdout.write(setup_cfg)


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print(f"usage: {sys.argv[0]} PYTHON-PKG-TAG")
        sys.exit(2)
    main(sys.argv[1])
