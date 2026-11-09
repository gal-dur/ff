#!/bin/sh
# The version ff reports (wyrd.wyrd `_version`): `git describe --tags --always`, as the Makefile
# computes it, over the repository alone. In: /in/repo/.git. Out: /out/version.json.
set -eu
v=$(git --git-dir=/in/repo/.git describe --tags --always)
printf '"%s"\n' "$v" >/out/version.json
