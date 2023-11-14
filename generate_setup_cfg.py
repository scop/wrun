#!/usr/bin/env python3

import re
import string
import sys
from urllib.parse import quote as urlquote
from urllib.request import urlopen

# TODO: verify linux/386 (i386), linux/armv6hf, and windows/386 markers
setup_cfg_template = """
[metadata]
name = wrun_py
version = ${pkg_version}
description = Python wrapper around invoking wrun (https://github.com/scop/wrun)
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
    Programming Language :: Python :: 3
    Topic :: Internet :: WWW/HTTP
    Topic :: Utilities

[options]
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
    sha256 = ${url_linux_armv6}
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


def main(pkg_version: str, checksums_txt_url: str) -> None:
    data = process_checksums(checksums_txt_url)
    data["pkg_version"] = pkg_version.lstrip("v")
    st = string.Template(setup_cfg_template.lstrip())
    setup_cfg = st.substitute(data)
    sys.stdout.write(setup_cfg)


if __name__ == "__main__":
    if len(sys.argv) != 3:
        print(f"usage: {sys.argv[0]} PKG-VERSION CHECKSUMS-TXT-URL")
        sys.exit(2)
    main(sys.argv[1], sys.argv[2])
