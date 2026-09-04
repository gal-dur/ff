# ff

Stage everything, write the commit message with a local model, commit. One binary,
no daemon: the pinned llama.cpp runtime and a 3B coder model are fetched into the
cache on first run and it works offline after that.

```
ff             stage, generate, commit
ff --dry-run   stage, print the message, commit nothing
```

`make build` (container-run Go, darwin/arm64 out), `make test`, `make install`
(into `$HOME/bins` — putting that on PATH is yours). The design, the pinned
artifacts and the env knobs are in [SPEC.md](SPEC.md).
