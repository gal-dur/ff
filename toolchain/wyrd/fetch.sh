#!/bin/sh
# The llama.cpp archive ff embeds (wyrd.wyrd `runtime`), fetched and verified by ff's own
# `cmd/pin fetch` exactly as the Makefile runs it, so the checksum dance lives in Go alone.
# `pin fetch` writes into the source tree, which is read-only under /in: it runs in a copy.
# In: /in/src, $GOOS, $GOARCH. Out: /out/runtime.tar.gz.
set -eu
cp -r /in/src /tmp/src
cd /tmp/src
go run ./cmd/pin fetch "$GOOS" "$GOARCH"
cp internal/provision/runtime.tar.gz /out/runtime.tar.gz
