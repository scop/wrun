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

import hashlib
from urllib.request import urlopen
import xml.etree.ElementTree as ET

ATOM_XMLNS = "http://www.w3.org/2005/Atom"


def latest_atom_version(url: str) -> str:
    with urlopen(url) as f:
        data = ET.parse(f)
    # For GitHub releases, title might differ from tag, look up from id
    id = data.find(f"{{{ATOM_XMLNS}}}entry/{{{ATOM_XMLNS}}}id").text
    # Sloppy, expected e.g. tag:github.com,2008:Repository/6731432/v0.10.0
    tag = id.rpartition("/")[2]
    return tag


def latest_rss2_version(url: str) -> str:
    with urlopen(url) as f:
        data = ET.parse(f)
    version = data.find("channel/item/title").text
    return version


def check_hexdigest(expected: str, algo: str, url: str | None) -> None:
    try:
        assert len(expected) == len(hashlib.new(algo, b"canary").hexdigest())
        _ = bytes.fromhex(expected)
    except Exception as e:
        raise ValueError(f'not a {algo} hex digest: "{expected}"') from e
    if not url:
        return
    with urlopen(url) as f:
        got = file_digest(f, algo).hexdigest()
    if got != expected:
        raise ValueError(f'{algo} mismatch for "{url}", expected {expected}, got {got}')


try:
    from hashlib import file_digest
except ImportError:  # Python < 3.11

    def file_digest(fileobj, digest: str):
        do = hashlib.new(digest)
        while data := fileobj.read(50 * 1024):
            do.update(data)
        return do
