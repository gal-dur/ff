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

# The bundled llama.cpp archive, fetched against the pins the code itself declares
# (cmd/pin prints them — one definition, no drift) and verified before it may be
# embedded. The pin differs per target, so the recipe always runs: a blob already
# matching the target's checksum is kept, anything else is refetched.
RUNTIME_BLOB := internal/provision/runtime.tar.gz

.PHONY: runtime-blob
runtime-blob:
	@eval "$$($(RUN) $(IMAGE) go run ./cmd/pin $(GOOS) $(GOARCH))"; \
	echo "$$RUNTIME_SHA256  $(RUNTIME_BLOB)" | shasum -a 256 -c - >/dev/null 2>&1 || { \
	  curl -fsSL "$$RUNTIME_URL" -o "$(RUNTIME_BLOB)"; \
	  echo "$$RUNTIME_SHA256  $(RUNTIME_BLOB)" | shasum -a 256 -c - >/dev/null \
	    || { rm -f "$(RUNTIME_BLOB)"; echo "runtime blob failed its checksum"; exit 1; }; \
	}

# Cross-compiled from the linux container for whatever target was asked for (pure Go,
# so that is a flag and not a toolchain). `embedruntime` bakes the llama.cpp archive
# in, so the shipped ff is one file and the only thing it ever downloads is the model.
build: runtime-blob
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
	rm -rf bin $(RUNTIME_BLOB)
