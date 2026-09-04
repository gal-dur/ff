# Host toolchains stay off this machine: Go runs in a container, with the module and
# build caches on a named volume so a warm build fetches and compiles nothing twice.
IMAGE := docker.io/library/golang:1.27
RUN   := podman run --rm -v $(PWD):/src -v ff-go:/go -w /src -e CGO_ENABLED=0

.PHONY: build test install clean

# The binary is darwin/arm64 — a personal tool for this machine, cross-compiled from
# the linux container (pure Go, so that is a flag and not a toolchain).
build:
	$(RUN) -e GOOS=darwin -e GOARCH=arm64 $(IMAGE) \
	  go build -trimpath -ldflags="-s -w" -o bin/ff ./cmd/ff

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
	rm -rf bin
