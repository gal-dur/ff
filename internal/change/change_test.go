package change

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// A throwaway repo with every file class the shaper distinguishes: a real change, a
// lockfile, a tracked file the repo's own ignore rules disown, and a binary.
func repo(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "t@example.test"},
		{"config", "user.name", "t"},
		{"commit", "-q", "--allow-empty", "-m", "init"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("calc.py", "def add(a, b):\n    return a + b\n")
	write("package-lock.json", `{"lockfileVersion": 3, "secretively-huge": true}`)
	// Tracked before its ignore rule landed: still in diffs, still machine junk.
	write("build.cache", "generated v1\n")
	if out, err := exec.Command("git", "add", "-f", "build.cache").CombinedOutput(); err != nil {
		t.Fatalf("add -f: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "commit", "-q", "-m", "track cache").CombinedOutput(); err != nil {
		t.Fatalf("commit: %v\n%s", err, out)
	}
	write("build.cache", "generated v2\n")
	write(".gitignore", "build.cache\n")
	write("logo.png", "\x89PNG\r\n\x1a\nnot really a png")
}

func TestShapedBudgetsTheSignalAndNamesTheNoise(t *testing.T) {
	repo(t)
	if err := Stage(); err != nil {
		t.Fatalf("stage: %v", err)
	}
	shaped, err := Shaped()
	if err != nil {
		t.Fatalf("shaped: %v", err)
	}

	if !strings.Contains(shaped, "Change overview") ||
		!strings.Contains(shaped, "Status per file") {
		t.Fatal("the overview or the file list is missing")
	}
	if !strings.Contains(shaped, "return a + b") {
		t.Error("the real change's patch is missing")
	}
	for _, junk := range []string{"package-lock.json", "build.cache", "logo.png"} {
		if !strings.Contains(shaped, junk+": machine-written, content omitted") {
			t.Errorf("%s was not content-omitted", junk)
		}
	}
	if strings.Contains(shaped, "secretively-huge") {
		t.Error("a lockfile's content leaked into the prompt")
	}
	if strings.Contains(shaped, "generated v2") {
		t.Error("a gitignore-matched tracked file's content leaked into the prompt")
	}
}

func TestNothingStagedIsEmptyNotAnError(t *testing.T) {
	repo(t)
	// Nothing staged: the tree is dirty but the index clean.
	shaped, err := Shaped()
	if err != nil {
		t.Fatalf("shaped: %v", err)
	}
	if shaped != "" {
		t.Fatalf("shaped = %q, want empty", shaped)
	}
}

func TestOneHugeFileCannotStarveTheRest(t *testing.T) {
	repo(t)
	big := strings.Repeat("x = 1\n", 10000)
	if err := os.WriteFile("aaa_first.py", []byte(big), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Stage(); err != nil {
		t.Fatalf("stage: %v", err)
	}
	shaped, err := Shaped()
	if err != nil {
		t.Fatalf("shaped: %v", err)
	}
	if !strings.Contains(shaped, "aaa_first.py truncated") {
		t.Error("the huge file was not truncated")
	}
	// The file sorting after the huge one still gets its patch seen whole.
	if !strings.Contains(shaped, "return a + b") {
		t.Error("a later file was starved by an earlier huge one")
	}
}
