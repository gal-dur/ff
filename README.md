# ff

Stage everything, write the commit message with a local model, commit. One binary,
no daemon: the pinned llama.cpp runtime and a 3B coder model are fetched into the
cache on first run and it works offline after that.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/gurunars/ff/master/install.sh | sh
```

Installs the latest release into `$HOME/bins` — putting that on PATH is yours.
Every push to master publishes a release named by `git describe`
(`.github/workflows/release.yml`).

## Use

```
ff             stage, generate, commit
ff --dry-run   stage, print the message, commit nothing
ff --version   which release this is
```

## Develop

`make build` (container-run Go, darwin/arm64 out), `make test`, `make install`
(local build into `$HOME/bins`). The design, the pinned artifacts and the env knobs
are in [SPEC.md](SPEC.md).
