// Package change stages the working tree and describes the staged change within a
// budget a small model can actually read whole.
package change

import (
	"fmt"
	"os/exec"
	"path"
	"strings"
)

// Character budgets: the overview and file list always arrive whole; patches are
// doled out per file, so a giant file cannot starve the rest — the failure the old
// single-head-truncation had.
const (
	totalBudget   = 24000
	perFileBudget = 4000
)

// Machine-written or non-text files: their *names* matter to the message, their
// hunks never do. The repo's own .gitignore extends this list at run time.
var noisy = []string{
	// Lockfiles and dependency manifests nobody edits by hand.
	"*.lock", "*-lock.json", "*-lock.yaml", "*.lockb", "go.sum", "go.work.sum",
	// Generated code and build output.
	"*.min.js", "*.min.css", "*.map", "*.pb.go", "*_pb2.py", "*_pb2.pyi",
	"*_pb2_grpc.py", "*.snap", "dist/*", "*/dist/*", "vendor/*", "*/vendor/*",
	"node_modules/*", "*/node_modules/*",
	// Non-text artifacts: the diff says "binary files differ", which says everything.
	"*.png", "*.jpg", "*.jpeg", "*.gif", "*.webp", "*.ico", "*.svg", "*.pdf",
	"*.woff", "*.woff2", "*.ttf", "*.otf", "*.eot",
	"*.zip", "*.gz", "*.tar", "*.jar", "*.wasm", "*.so", "*.dylib", "*.a", "*.bin",
	"*.mp3", "*.mp4", "*.sqlite", "*.db",
}

func git(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}

// Stage stages the whole repository. -A, not `.`: everything means everything,
// whichever subdirectory ff was run from.
func Stage() error {
	_, err := git("add", "-A")
	return err
}

func isNoisy(file string) bool {
	for _, pattern := range noisy {
		if ok, _ := path.Match(pattern, file); ok {
			return true
		}
	}
	return false
}

// gitignored answers which of the paths the repo's own ignore rules call machine
// junk. Tracked files are exempt from ignore rules, so files committed before their
// ignore rule landed still turn up in diffs — `--no-index` asks what the *patterns*
// say regardless of tracking. Exit code 1 means "none matched": an answer, not an
// error.
func gitignored(paths []string) map[string]bool {
	command := exec.Command("git", "check-ignore", "--no-index", "--stdin")
	command.Stdin = strings.NewReader(strings.Join(paths, "\n"))
	out, _ := command.Output()
	ignored := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		if line != "" {
			ignored[line] = true
		}
	}
	return ignored
}

// Shaped is the staged change as the prompt carries it, or "" when there is none.
func Shaped() (string, error) {
	stat, err := git("diff", "--cached", "--stat=100")
	if err != nil {
		return "", err
	}
	nameStatus, err := git("diff", "--cached", "--name-status")
	if err != nil {
		return "", err
	}
	nameStatus = strings.TrimSpace(nameStatus)
	if nameStatus == "" {
		return "", nil
	}

	var paths []string
	for _, line := range strings.Split(nameStatus, "\n") {
		fields := strings.Split(line, "\t")
		paths = append(paths, fields[len(fields)-1])
	}
	ignored := gitignored(paths)

	var patches []string
	spent := 0
	for _, file := range paths {
		switch {
		case isNoisy(file) || ignored[file]:
			patches = append(patches, fmt.Sprintf("--- %s: machine-written, content omitted", file))
		case spent >= totalBudget:
			patches = append(patches, fmt.Sprintf("--- %s: diff omitted, budget spent", file))
		default:
			patch, err := git("diff", "--cached", "--", file)
			if err != nil {
				return "", err
			}
			if len(patch) > perFileBudget {
				patch = patch[:perFileBudget] + fmt.Sprintf("\n... [%s truncated]\n", file)
			}
			patches = append(patches, patch)
			spent += len(patch)
		}
	}

	return strings.Join([]string{
		"Change overview (files and line counts):",
		strings.TrimSpace(stat),
		"",
		"Status per file (A added, M modified, D deleted, R renamed):",
		nameStatus,
		"",
		"Patches:",
		strings.Join(patches, "\n"),
	}, "\n"), nil
}
