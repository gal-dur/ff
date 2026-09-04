// Package pin names the exact artifacts ff runs on — the one place a version bump or
// a second platform would be added. The build scripts read these through cmd/pin, so
// the Makefile and the release workflow can never disagree with the code.
package pin

const (
	RuntimeURL = "https://github.com/ggml-org/llama.cpp/releases/download/b10797/" +
		"llama-b10797-bin-macos-arm64.tar.gz"
	RuntimeSHA256 = "474a788ec73d17a066360b1c50c9733c78a47d062616e91963c65a344548e889"
	// The archive's top-level directory, and its file name in the cache.
	RuntimeDir     = "llama-b10797"
	RuntimeArchive = "llama-b10797-bin-macos-arm64.tar.gz"

	ModelURL = "https://huggingface.co/Qwen/Qwen2.5-Coder-3B-Instruct-GGUF/resolve/main/" +
		"qwen2.5-coder-3b-instruct-q4_k_m.gguf"
	ModelSHA256 = "724fb256bec1ff062b2f65e4569e871ad2e95ab2a3989723d1769c54294730b7"
	ModelFile   = "qwen2.5-coder-3b-instruct-q4_k_m.gguf"
)
