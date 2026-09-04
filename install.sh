#!/bin/sh
# Install the latest released ff into $HOME/bins. Putting that on PATH is yours —
# this script never edits a shell profile; it only says when it would matter.
set -eu

mkdir -p "$HOME/bins"
curl -fsSL "https://github.com/gal-dur/ff/releases/latest/download/ff" \
  -o "$HOME/bins/ff"
chmod +x "$HOME/bins/ff"
echo "ff: installed $("$HOME/bins/ff" --version 2>/dev/null || echo ff) to $HOME/bins/ff"
case ":$PATH:" in
  *":$HOME/bins:"*) ;;
  *) echo "ff: note — \$HOME/bins is not on your PATH" ;;
esac
