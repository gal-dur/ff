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

mkdir -p "$HOME/bins"
curl -fsSL "https://github.com/gal-dur/ff/releases/latest/download/ff-$goos-$goarch" \
  -o "$HOME/bins/ff"
chmod +x "$HOME/bins/ff"
echo "ff: installed $("$HOME/bins/ff" --version 2>/dev/null || echo ff) to $HOME/bins/ff"
case ":$PATH:" in
  *":$HOME/bins:"*) ;;
  *) echo "ff: note — \$HOME/bins is not on your PATH" ;;
esac
