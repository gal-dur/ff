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

	// 7B over 3B, measured on a real diff: the right commit type, accurate bullets,
	// and it caught changes the 3B missed, at ~1.5x the run time. The next size down
	// is the knob to turn if latency ever outweighs message quality.
	ModelURL = "https://huggingface.co/Qwen/Qwen2.5-Coder-7B-Instruct-GGUF/resolve/main/" +
		"qwen2.5-coder-7b-instruct-q4_k_m.gguf"
	ModelSHA256 = "509287f78cb4d4cf6b3843734733b914b2c158e43e22a7f4bf5e963800894d3c"
	ModelFile   = "qwen2.5-coder-7b-instruct-q4_k_m.gguf"
)
