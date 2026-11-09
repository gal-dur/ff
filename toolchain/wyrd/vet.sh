#!/bin/sh
# go vet over the module, as one std.LintReport (wyrd.wyrd `vet`): findings are a value, so the
# script exits 0 once report.json is written. In: /in/src. Out: /out/report.json, /out/vet.log.
set -u
cd /in/src
go vet ./... >/out/vet.log 2>&1
code=$?
findings=$(grep -cE '^[^#].*\.go:[0-9]+:[0-9]+: ' /out/vet.log)
first=$(grep -m1 -E '^[^#].*\.go:[0-9]+:[0-9]+: ' /out/vet.log | sed 's/["\\]/ /g')
if [ "$code" -eq 0 ]; then
  printf '{"ok":true,"findings":0}\n' >/out/report.json
elif [ "$findings" -gt 0 ]; then
  printf '{"ok":false,"message":"%d findings; first: %s","findings":%d}\n' "$findings" "$first" "$findings" >/out/report.json
else
  printf '{"ok":false,"message":"go vet exited %d: %s","findings":0}\n' "$code" "$(head -n1 /out/vet.log | sed 's/["\\]/ /g')" >/out/report.json
fi
