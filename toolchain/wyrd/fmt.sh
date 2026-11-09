#!/bin/sh
# gofmt over the module (wyrd.wyrd `fmt`): it rewrites a private, writable copy of the sources
# in place, and wyrd makes the patch of what changed. The files it rewrote are the findings.
# In: /in/src (edit<Files>). Out: /out/report.json.
set -eu
cd /in/src
changed=$(gofmt -l .)
gofmt -w .
n=$(printf '%s' "$changed" | grep -c . || true)
if [ "$n" -eq 0 ]; then
  printf '{"ok":true,"findings":0}\n' >/out/report.json
else
  printf '{"ok":false,"message":"%d files not gofmt-formatted; first: %s","findings":%d}\n' \
    "$n" "$(printf '%s\n' "$changed" | head -n1)" "$n" >/out/report.json
fi
