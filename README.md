# ff

Stage everything, write the commit message with a local model, commit. One binary,
no daemon: the pinned llama.cpp runtime rides inside the binary and self-extracts on
first run; the 3B coder model is fetched into the cache once. Offline after that.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/gal-dur/ff/master/install.sh | sh
```

Installs the latest release into `$HOME/bins` — putting that on PATH is yours.
Releases are cut by pushing a `vX.Y.Z` tag (`.github/workflows/release.yml`);
the tag is both the trigger and the version the binary reports.

## Use

```
ff             stage, generate, commit
ff --dry-run   stage, print the message, commit nothing
ff --version   which release this is
```

First run extracts the bundled runtime and fetches the pinned model into the cache,
checksum-verified; every later run is offline. All knobs are env vars, all optional:

| var | meaning |
|---|---|
| `FF_CACHE_DIR` | cache location (default: the OS user cache dir + `/ff`) |
| `FF_MODEL_FILE` | path to your own GGUF, skipping the pinned download |
| `FF_CTX` | context window (default 16384) |
| `FF_MAX_TOKENS` | generation cap (default 400) |

## Develop

`make build` (container-run Go, darwin/arm64 out), `make test`, `make install`
(local build into `$HOME/bins`). The artifact pins live in
`internal/provision/provision.go` — the one place a version bump or a second
platform would be added; design rationale lives as comments on the code it governs.
