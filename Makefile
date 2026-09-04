# Host toolchains stay off this machine: Go runs in a container, with the module and
# build caches on a named volume so a warm build fetches and compiles nothing twice.
IMAGE := docker.io/library/golang:1.27
RUN   := podman run --rm -v $(PWD):/src -v ff-go:/go -w /src -e CGO_ENABLED=0

.PHONY: build test install clean

# The version is git's own description, computed on the host (the container's git
# would balk at the mounted repo's ownership) and injected the same way the release
# workflow injects it.
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

# The target platform: this machine's, unless overridden (`make build GOOS=linux
# GOARCH=arm64`). The pinned platforms are declared in internal/pin.
GOOS   ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
GOARCH ?= $(shell uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/')

# The bundled llama.cpp archive, fetched and verified by `cmd/pin fetch` — the one
# definition of the fetch-then-verify dance lives in Go, not here. The stamp keeps
# warm builds free of container spawns; it names the target because the blob's right
# content differs per platform, and it depends on the pins so a bump refetches.
RUNTIME_BLOB := internal/provision/runtime.tar.gz
BLOB_STAMP   := internal/provision/.runtime-blob-$(GOOS)-$(GOARCH)

$(BLOB_STAMP): internal/pin/pin.go
	$(RUN) $(IMAGE) go run ./cmd/pin fetch $(GOOS) $(GOARCH)
	@rm -f internal/provision/.runtime-blob-* && touch $@

# Cross-compiled from the linux container for whatever target was asked for (pure Go,
# so that is a flag and not a toolchain). `embedruntime` bakes the llama.cpp archive
# in, so the shipped ff is one file and the only thing it ever downloads is the model.
build: $(BLOB_STAMP)
	$(RUN) -e GOOS=$(GOOS) -e GOARCH=$(GOARCH) $(IMAGE) \
	  go build -trimpath -tags embedruntime \
	  -ldflags="-s -w -X main.version=$(VERSION)" -o bin/ff ./cmd/ff

test:
	$(RUN) $(IMAGE) go test ./...

# One fixed, user-owned destination: $HOME/bins. Putting it on PATH is the user's
# call, and the install says so rather than editing anyone's shell profile.
install: build
	@mkdir -p "$$HOME/bins" && cp bin/ff "$$HOME/bins/ff" && \
	  echo "ff: installed to $$HOME/bins/ff"; \
	case ":$$PATH:" in \
	  *":$$HOME/bins:"*) ;; \
	  *) echo "ff: note — $$HOME/bins is not on your PATH";; \
	esac

clean:
	rm -rf bin $(RUNTIME_BLOB) internal/provision/.runtime-blob-*
