#!/bin/sh
# ff's release binary (wyrd.wyrd `build`): the Makefile's `go build -tags embedruntime`, with the
# runtime archive `fetch` produced placed where runtime_embed.go's go:embed reads it.
# In: /in/src, /in/runtime, $GOOS, $GOARCH, $VERSION. Out: /out/bin/ff.
set -eu
cp -r /in/src /tmp/src
cp /in/runtime /tmp/src/internal/provision/runtime.tar.gz
cd /tmp/src
mkdir -p /out/bin
go build -trimpath -tags embedruntime -ldflags="-s -w -X main.version=$VERSION" -o /out/bin/ff ./cmd/ff
