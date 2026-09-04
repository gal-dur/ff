package message

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanStripsEveryWrapperTheModelInvents(t *testing.T) {
	for name, in := range map[string]struct{ raw, want string }{
		"already clean": {"feat: add thing\n\n- because\n", "feat: add thing\n\n- because"},
		"fenced":        {"```\nfix: a bug\n```\n", "fix: a bug"},
		"fenced with lang": {"```text\nfix: a bug\n```", "fix: a bug"},
		"preamble":      {"Here's the commit message:\nchore: tidy\n", "chore: tidy"},
		"quoted":        {`"docs: explain"`, "docs: explain"},
		"end marker":    {"perf: faster\n[end of text]", "perf: faster"},
	} {
		if got := Clean(in.raw); got != in.want {
			t.Errorf("%s: Clean(%q) = %q, want %q", name, in.raw, got, in.want)
		}
	}
}

// A stub standing in for llama-cli: the tests prove the invocation and parsing, the
// real binary is proven by using the tool.
func stub(t *testing.T, script string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "llama-cli")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestGenerateRunsOneTurnAndCleansTheAnswer(t *testing.T) {
	runtime := stub(t, "echo '```'\necho 'feat: stubbed message'\necho '```'")
	msg, err := Generate(runtime, "model.gguf", "the diff")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if msg != "feat: stubbed message" {
		t.Fatalf("msg = %q", msg)
	}
}

func TestAFailingRuntimeSurfacesItsLastWords(t *testing.T) {
	runtime := stub(t, `echo "ggml_metal: out of memory" >&2; exit 1`)
	_, err := Generate(runtime, "model.gguf", "the diff")
	if err == nil || !strings.Contains(err.Error(), "out of memory") {
		t.Fatalf("err = %v, want the runtime's stderr tail", err)
	}
}

func TestAnEmptyAnswerIsAnError(t *testing.T) {
	runtime := stub(t, `true`)
	if _, err := Generate(runtime, "model.gguf", "the diff"); err == nil {
		t.Fatal("an empty response was accepted")
	}
}
