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
wrun_shfmt_args.py -- generate wrun command line args for shfmt

* https://github.com/mvdan/sh#shfmt
* https://github.com/mvdan/sh/releases
"""

import hashlib
import sys
from urllib.parse import urljoin, quote as urlquote
from urllib.request import urlopen

file_os_archs = {
    "shfmt_{version}_darwin_amd64": "darwin/amd64",
    "shfmt_{version}_darwin_arm64": "darwin/arm64",
    "shfmt_{version}_linux_386": "linux/386",
    "shfmt_{version}_linux_amd64": "linux/amd64",
    "shfmt_{version}_linux_arm": "linux/arm",
    "shfmt_{version}_linux_arm64": "linux/arm64",
    "shfmt_{version}_windows_386.exe": "windows/386",
    "shfmt_{version}_windows_amd64.exe": "windows/amd64",
}


def main(version: str) -> None:
    base_url = f"https://github.com/mvdan/sh/releases/download/{urlquote(version)}/"

    for fn, os_arch in file_os_archs.items():
        url = urljoin(base_url, urlquote(fn.format(version=version)))
        with urlopen(url) as f:
            digest = hashlib.file_digest(f, "sha256")

        print(f"-url {os_arch}={url}#sha256-{digest.hexdigest()}")


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print(f"usage: {sys.argv[0]} VERSION")
        sys.exit(2)
    main(sys.argv[1])
