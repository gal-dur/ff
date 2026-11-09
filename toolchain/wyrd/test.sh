#!/bin/sh
# ff's tests under coverage, as one std.TestReport (wyrd.wyrd `test`). A failing test is a
# value: the script exits 0 once report.json is written, red or green; a package that does not
# compile is red too, named in the message.
# In: /in/src (the module), $RUN (optional -run filter). Out: /out/report.json, /out/test.log.
set -u
cd /in/src
out=/out
json=/tmp/test.json
go test -json -count=1 -coverprofile=/tmp/cover.out ${RUN:+-run "$RUN"} ./... >"$json" 2>/tmp/stderr
code=$?
# The human log: every test's output lines, as `go test` prints them, then go's stderr.
sed -n 's/.*"Action":"output".*"Output":"\(.*\)"}$/\1/p' "$json" | sed 's/\\n$//; s/\\t/\t/g; s/\\"/"/g' >"$out/test.log"
cat /tmp/stderr >>"$out/test.log"
passed=$(grep -c '"Action":"pass","Package":"[^"]*","Test":' "$json")
failed=$(grep -c '"Action":"fail","Package":"[^"]*","Test":' "$json")
# A package that does not build fails without a test failing: count it as a failure.
broken=$(grep -c '"Action":"fail","Package":"[^"]*","Elapsed"' "$json")
first=$(sed -n 's/.*"Action":"fail","Package":"\([^"]*\)","Test":"\([^"]*\)".*/\1.\2/p' "$json" | head -n 1)
lines=0 covered=0
if [ -s /tmp/cover.out ]; then
  # Statements, each block counted once across packages that share it (-coverprofile merges).
  set -- $(awk 'NR > 1 { n[$1] = $2; if ($3 > 0) hit[$1] = 1 }
               END { for (k in n) { t += n[k]; if (k in hit) c += n[k] } printf "%d %d", t, c }' /tmp/cover.out)
  lines=$1 covered=$2
fi
tests=$((passed + failed))
if [ "$failed" -eq 0 ] && [ "$broken" -eq 0 ] && [ "$code" -eq 0 ]; then
  ok=true message=""
elif [ -n "$first" ]; then
  ok=false message=",\"message\":\"$failed failed; first: $first\""
else
  ok=false message=",\"message\":\"go test exited $code: $(grep -m1 -v '^#' /tmp/stderr | sed 's/["\\]/ /g')\""
fi
coverage=""
if [ "$lines" -gt 0 ]; then
  coverage=$(awk -v c="$covered" -v l="$lines" 'BEGIN { printf ",\"coverage\":%.6f", c / l }')
fi
printf '{"ok":%s%s,"tests":%d,"passed":%d,"failed":%d,"lines":%d,"covered":%d%s}\n' \
  "$ok" "$message" "$tests" "$passed" "$failed" "$lines" "$covered" "$coverage" >"$out/report.json"
