# ff — a self-contained commit-message tool

One command: stage everything, write the commit message with a local model, commit.
A single Go binary that owns its own inference — no Ollama, no daemon, no service to
have forgotten to start. After the first run it works offline.

## Why a rewrite

The Python/Ollama version produced degraded messages for two mechanical reasons that
this design removes rather than patches:

- **Front-truncation ate the instructions.** Ollama's default context is ~2k tokens
  and it truncates an overflowing prompt from the front — precisely where the rules
  sat. Here the context size is explicit, the rules sit *after* the diff, and the
  diff itself is budgeted per file so it cannot overflow in the first place.
- **A daemon is a dependency with moods.** "Is ollama running", "which model did it
  load", "what did an upgrade change" — all gone. The runtime and the model are
  pinned artifacts this tool provisions itself, by checksum.

## Behaviour

```
ff            stage everything, print the generated message, commit with it
ff --dry-run  stage everything, print the message, commit nothing
```

Commits use `--no-verify`, matching the tool this replaces. Exit codes: 0 on commit
(or clean dry-run), non-zero with a one-line error otherwise.

## Self-provisioning

First run downloads two artifacts into the cache (`os.UserCacheDir()/ff`, override
with `FF_CACHE_DIR`); every later run just uses them.

| artifact | pinned | size | sha256 |
|---|---|---|---|
| llama.cpp runtime | `b10797`, `llama-b10797-bin-macos-arm64.tar.gz` | 11 MB | `474a788ec73d17a066360b1c50c9733c78a47d062616e91963c65a344548e889` |
| model | `Qwen/Qwen2.5-Coder-3B-Instruct-GGUF`, `q4_k_m` | 2.0 GB | `724fb256bec1ff062b2f65e4569e871ad2e95ab2a3989723d1769c54294730b7` |

- Downloads land as `.partial`, are verified against the pinned sha256, and only then
  renamed into place — a torn download or a swapped upstream file cannot be used.
  No resume in v1: an interrupted 2 GB download restarts.
- The runtime tarball is extracted by a pure-Go extractor that preserves the exec
  bit and refuses entries that would escape the destination directory.
- 3B is the deliberate default: for commit messages its output is indistinguishable
  from 7B, and it loads in about a second. `FF_MODEL_FILE=/path/to/model.gguf`
  points at any other local GGUF; the tool never re-verifies a user-supplied file.

## The change, shaped for a reader with a budget

Ported intact from the proven Python version:

- `git add .`, then the *staged* change is described three ways: a `--stat` overview
  and a `--name-status` list always arrive whole; per-file patches are capped
  (4,000 chars each, 24,000 total) so one huge file cannot starve the rest — the
  failure the old single-head-truncation had.
- **Noisy files are named, never dissected**: lockfiles, generated code, and
  non-text artifacts by glob, plus whatever the repo's own ignore rules match —
  `git check-ignore --no-index` applies `.gitignore` patterns even to tracked files,
  which catches files committed before their ignore rule landed.
- The rules follow the diff in the prompt, so if anything ever truncates it is diff
  that goes, never instructions.

## Inference

`llama-cli --single-turn` per invocation: no server, no port, no state between runs.
Temperature 0.2, context 16384, generation capped at 400 tokens, all layers offloaded
to Metal. The runtime's logs go to stderr and are discarded unless it fails; stdout
is the message. The response is then cleaned — code fences, wrapping quotes and
"here's the commit message:" preambles stripped — and committed verbatim.

Environment knobs, all optional: `FF_CACHE_DIR`, `FF_MODEL_FILE`, `FF_CTX`,
`FF_MAX_TOKENS`.

## Layout

```
cmd/ff/            the binary: flag parsing and the sequence, nothing else
internal/change/   staging and the budgeted change summary
internal/provision/ checksum-verified downloads, tar.gz extraction, cache layout
internal/message/  prompt assembly, llama-cli invocation, response cleanup
```

## Building and testing

Host toolchains stay off this machine: the Makefile runs Go inside a container
(`golang:1.27`), with the module and build caches on a named volume so warm builds
install nothing. The binary cross-compiles to darwin/arm64 (pure Go, `CGO_ENABLED=0`).

```
make build     bin/ff (darwin/arm64)
make test      the suite, in the container
make install   bin/ff into $HOME/bins, printed; PATH is the user's business
```

`install` writes to one fixed, user-owned place and says so. It never edits a shell
profile; it notes when `$HOME/bins` is not on `PATH` and leaves the decision there.

Tests need no network and no model: the change summary runs against throwaway git
repos, provisioning against an in-test HTTP server serving in-test tarballs, and
inference against a stub executable standing in for llama-cli.

## Non-goals

- No resume of partial downloads, no model management UI, no config file: pinned
  defaults and four env vars are the whole surface.
- macOS arm64 only, deliberately — it is a personal tool; the pinning table is the
  one place a second platform would be added.
- Never a daemon. The day per-run model loading annoys, the answer is a smaller
  model, not a server.
