#!/bin/sh
# Install the latest released ff into $HOME/bins. Putting that on PATH is yours —
# this script never edits a shell profile; it only says when it would matter.
set -eu

# Release assets are named ff-<goos>-<goarch>; uname maps onto that here.
case "$(uname -s)" in
  Darwin) goos=darwin ;;
  Linux)  goos=linux ;;
  *) echo "ff: unsupported OS: $(uname -s) (darwin and linux only)" >&2; exit 1 ;;
esac
case "$(uname -m)" in
  x86_64)        goarch=amd64 ;;
  arm64|aarch64) goarch=arm64 ;;
  *) echo "ff: unsupported architecture: $(uname -m) (amd64 and arm64 only)" >&2; exit 1 ;;
esac
asset="ff-$goos-$goarch"
base="https://github.com/gal-dur/ff/releases/latest/download"

# Verified before installed, against the SHA256SUMS the release workflow publishes —
# the same pin-then-check treatment ff gives its own artifacts.
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
curl -fsSL "$base/$asset" -o "$tmp/$asset"
curl -fsSL "$base/SHA256SUMS" -o "$tmp/SHA256SUMS"
if command -v sha256sum >/dev/null 2>&1; then sum="sha256sum"; else sum="shasum -a 256"; fi
(cd "$tmp" && grep " $asset\$" SHA256SUMS | $sum -c - >/dev/null) \
  || { echo "ff: $asset failed its checksum; not installing" >&2; exit 1; }

mkdir -p "$HOME/bins"
install -m 0755 "$tmp/$asset" "$HOME/bins/ff"
echo "ff: installed $("$HOME/bins/ff" --version 2>/dev/null || echo ff) to $HOME/bins/ff"
case ":$PATH:" in
  *":$HOME/bins:"*) ;;
  *) echo "ff: note — \$HOME/bins is not on your PATH" ;;
esac
