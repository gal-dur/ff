// Package pin names the exact artifacts ff runs on — the one place a version bump or
// another platform would be added. The build scripts read these through cmd/pin, so
// the Makefile and the release workflow can never disagree with the code.
package pin

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

const (
	runtimeRelease = "b10797"
	runtimeBase    = "https://github.com/ggml-org/llama.cpp/releases/download/" +
		runtimeRelease + "/"
	// The archives all unpack to the same top-level directory.
	RuntimeDir = "llama-" + runtimeRelease
)

// Artifact is one pinned download. Name doubles as the file's name in the cache.
type Artifact struct {
	URL, SHA256, Name string
}

// Model is the GGUF ff generates with. 7B over 3B, measured on a real diff: the
// right commit type, accurate bullets, and it caught changes the 3B missed, at
// ~1.5x the run time. The next size down is the knob to turn if latency ever
// outweighs message quality.
var Model = Artifact{
	URL: "https://huggingface.co/Qwen/Qwen2.5-Coder-7B-Instruct-GGUF/resolve/main/" +
		"qwen2.5-coder-7b-instruct-q4_k_m.gguf",
	SHA256: "509287f78cb4d4cf6b3843734733b914b2c158e43e22a7f4bf5e963800894d3c",
	Name:   "qwen2.5-coder-7b-instruct-q4_k_m.gguf",
}

// The upstream CPU/Metal builds, one per platform ff ships for. Every checksum was
// computed from the downloaded asset, not copied from upstream.
var runtimes = map[string]Artifact{
	"darwin/arm64": runtime("macos-arm64",
		"474a788ec73d17a066360b1c50c9733c78a47d062616e91963c65a344548e889"),
	"darwin/amd64": runtime("macos-x64",
		"a12a85385c74e1e0260dd207cc49f90db902df0fa2f12fc734971d0323aa1df0"),
	"linux/amd64": runtime("ubuntu-x64",
		"8a61fe2b9f7c0e05058c375a9c52433241897d2f4b0170b4e8bb43acaa3319d9"),
	"linux/arm64": runtime("ubuntu-arm64",
		"2ecebe067cae4b8ceea858e0fbad793fca2cba5203acd039ccee564ae4ecd455"),
}

func runtime(platform, sha256 string) Artifact {
	name := "llama-" + runtimeRelease + "-bin-" + platform + ".tar.gz"
	return Artifact{URL: runtimeBase + name, SHA256: sha256, Name: name}
}

// Platforms answers the pinned GOOS/GOARCH pairs, sorted — every roster of what ff
// ships for derives from the map above through here.
func Platforms() []string {
	return slices.Sorted(maps.Keys(runtimes))
}

// RuntimeFor answers the pinned runtime for a GOOS/GOARCH pair.
func RuntimeFor(goos, goarch string) (Artifact, error) {
	r, ok := runtimes[goos+"/"+goarch]
	if !ok {
		return Artifact{}, fmt.Errorf("no pinned runtime for %s/%s (ff ships for %s)",
			goos, goarch, strings.Join(Platforms(), ", "))
	}
	return r, nil
}
