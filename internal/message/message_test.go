package message

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanStripsEveryWrapperTheModelInvents(t *testing.T) {
	for name, in := range map[string]struct{ raw, want string }{
		"already clean":    {"feat: add thing\n\n- because\n", "feat: add thing\n\n- because"},
		"fenced":           {"```\nfix: a bug\n```\n", "fix: a bug"},
		"fenced with lang": {"```text\nfix: a bug\n```", "fix: a bug"},
		"preamble":         {"Here's the commit message:\nchore: tidy\n", "chore: tidy"},
		"quoted":           {`"docs: explain"`, "docs: explain"},
		"end marker":       {"perf: faster\n[end of text]", "perf: faster"},
		"repetition loop": {
			"feat: thing\n\n- once\n- once\n- once\n- once\n- once\n- twice\n",
			"feat: thing\n\n- once\n- twice"},
		"runaway body": {
			"fix: bug\n\n- a\n- b\n- c\n- d\n- e\n- f\n",
			"fix: bug\n\n- a\n- b\n- c\n- d"},
	} {
		if got := Clean(in.raw); got != in.want {
			t.Errorf("%s: Clean(%q) = %q, want %q", name, in.raw, got, in.want)
		}
	}
}

func TestCompleteSpeaksTheChatAPI(t *testing.T) {
	var asked map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&asked)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"feat: answered"}}]}`))
	}))
	defer server.Close()

	got, err := complete(server.URL, "the prompt")
	if err != nil || got != "feat: answered" {
		t.Fatalf("complete = %q, %v", got, err)
	}
	messages := asked["messages"].([]any)
	content := messages[0].(map[string]any)["content"].(string)
	if content != "the prompt" {
		t.Fatalf("prompt sent = %q", content)
	}
}

func TestAnErrorStatusIsAnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "loading", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	if _, err := complete(server.URL, "x"); err == nil {
		t.Fatal("a 503 read as success")
	}
}

// The helper doubles as a fake llama-server via the classic re-exec pattern: the stub
// script execs this test binary, which serves /health and one completion.
func TestHelperLlamaServer(t *testing.T) {
	if os.Getenv("FF_HELPER") != "1" {
		t.Skip("helper process, not a test")
	}
	args := strings.Fields(os.Getenv("FF_HELPER_ARGS"))
	port := ""
	for i, arg := range args {
		if arg == "--port" && i+1 < len(args) {
			port = args[i+1]
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"` + "```" + `\nfeat: from helper\n` + "```" + `"}}]}`))
	})
	_ = http.ListenAndServe("127.0.0.1:"+port, mux) // killed by the test's stop()
}

// fakeRuntime lays out a directory the way the real cache does: a llama-cli path the
// caller holds, with llama-server beside it — here a script re-execing the test
// binary as TestHelperLlamaServer.
func fakeRuntime(t *testing.T, serverScript string) string {
	t.Helper()
	dir := t.TempDir()
	cli := filepath.Join(dir, "llama-cli")
	if err := os.WriteFile(cli, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "llama-server"), []byte(serverScript), 0o755); err != nil {
		t.Fatal(err)
	}
	return cli
}

func TestGenerateSpawnsAsksCleansAndStops(t *testing.T) {
	testBinary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cli := fakeRuntime(t, fmt.Sprintf(
		"#!/bin/sh\nexport FF_HELPER=1 FF_HELPER_ARGS=\"$*\"\nexec %q -test.run '^TestHelperLlamaServer$'\n",
		testBinary))

	msg, err := Generate(cli, "model.gguf", "the diff")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if msg != "feat: from helper" {
		t.Fatalf("msg = %q, want the fenced answer cleaned", msg)
	}
}

func TestAServerThatDiesSurfacesItsLastWords(t *testing.T) {
	cli := fakeRuntime(t, "#!/bin/sh\necho 'gguf: file corrupt' >&2\nexit 1\n")
	_, err := Generate(cli, "model.gguf", "the diff")
	if err == nil || !strings.Contains(err.Error(), "file corrupt") {
		t.Fatalf("err = %v, want the server's stderr tail", err)
	}
}
