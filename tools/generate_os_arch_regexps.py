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

import re
import subprocess


def main() -> None:
    oss = set()
    archs = set()

    proc = subprocess.run(
        ["go", "tool", "dist", "list"], capture_output=True, check=True, text=True
    )
    for line in proc.stdout.splitlines():
        os, sep, arch = line.partition("/")
        if sep != "":
            oss.add(os)
            archs.add(arch)

    os_alts = "|".join(re.escape(x) for x in sorted(oss))
    print(f"({os_alts})")
    arch_alts = "|".join(re.escape(x) for x in sorted(archs))
    print(f"({arch_alts})")


if __name__ == "__main__":
    main()
