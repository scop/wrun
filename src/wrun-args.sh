#!/bin/sh
set -eu
tool=${0##*/wrun-}
tool=${tool%-args}
PYTHONPATH=/usr/share/wrun exec python3 -m "wrun_py.generators.$tool" "$@"
