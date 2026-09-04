# ff

Stage everything, write the commit message with a local model, commit. One binary,
no daemon: the pinned llama.cpp runtime rides inside the binary and self-extracts on
first run; the 7B coder model is fetched into the cache once. Offline after that.
Runs on macOS and Linux (arm64 and amd64 each).

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/gal-dur/ff/master/install.sh | sh
```

Picks the release binary for your OS and architecture (by `uname`) and installs it
into `$HOME/bins` — putting that on PATH is yours.
Releases are cut by pushing a `vX.Y.Z` tag (`.github/workflows/release.yml`);
the tag is both the trigger and the version the binary reports.

## Use

```
ff             stage, generate, commit
ff --version   which release this is
```

Commit hooks run as they would for a hand-written message; ff does not bypass them.

First run extracts the bundled runtime and fetches the pinned model into the cache,
checksum-verified; every later run is offline. On Linux the runtime needs `libgomp`
and `libcurl` from the distro (present on any typical install; on a minimal Debian/
Ubuntu: `apt install libgomp1 libcurl4`). All knobs are env vars, all optional:

| var | meaning |
|---|---|
| `FF_CACHE_DIR` | cache location (default: the OS user cache dir + `/ff`) |
| `FF_MODEL_FILE` | path to your own GGUF, skipping the pinned download |
| `FF_CTX` | context window (default 16384) |
| `FF_MAX_TOKENS` | generation cap (default 400) |

## Develop

`make build` (container-run Go, a binary for this machine out — override with
`make build GOOS=linux GOARCH=arm64`), `make test`, `make install` (local build
into `$HOME/bins`). The artifact pins live in `internal/pin/pin.go` — the one
place a version bump or another platform would be added; design rationale lives
as comments on the code it governs.
