// Command ff stages everything, writes the commit message with a local model, and
// commits. See SPEC.md — the spec is the contract, this file is just the sequence.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gal-dur/ff/internal/change"
	"github.com/gal-dur/ff/internal/message"
	"github.com/gal-dur/ff/internal/provision"
)

// Injected at build time from `git describe` — see the Makefile and the release
// workflow, which must agree on the flag.
var version = "dev"

func fail(err error) {
	fmt.Fprintln(os.Stderr, "ff:", err)
	os.Exit(1)
}

func main() {
	dryRun := false
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--dry-run":
			dryRun = true
		case "--version":
			fmt.Println("ff", version)
			return
		default:
			fail(fmt.Errorf("unknown argument %q (usage: ff [--dry-run|--version])", arg))
		}
	}

	if err := exec.Command("git", "rev-parse", "--is-inside-work-tree").Run(); err != nil {
		fail(fmt.Errorf("not inside a git repository"))
	}
	if err := change.Stage(); err != nil {
		fail(err)
	}
	shaped, err := change.Shaped()
	if err != nil {
		fail(err)
	}
	if shaped == "" {
		fail(fmt.Errorf("nothing to commit after staging"))
	}

	cache := os.Getenv("FF_CACHE_DIR")
	if cache == "" {
		base, err := os.UserCacheDir()
		if err != nil {
			fail(err)
		}
		cache = filepath.Join(base, "ff")
	}
	runtime, err := provision.Runtime(cache)
	if err != nil {
		fail(fmt.Errorf("provision runtime: %w", err))
	}
	model, err := provision.Model(cache)
	if err != nil {
		fail(fmt.Errorf("provision model: %w", err))
	}

	msg, err := message.Generate(runtime, model, shaped)
	if err != nil {
		fail(err)
	}

	fmt.Println(msg)
	if dryRun {
		return
	}
	commit := exec.Command("git", "commit", "--no-verify", "-m", msg)
	commit.Stdout, commit.Stderr = os.Stdout, os.Stderr
	if err := commit.Run(); err != nil {
		fail(fmt.Errorf("git commit failed"))
	}
}
