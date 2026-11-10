package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/gal-dur/ff/internal/pin"
)

// The test binary doubles as pin itself: run with PIN_TEST_AS_MAIN set, it is the command,
// so the tests drive pin's real argument handling and exit codes.
func TestMain(m *testing.M) {
	if os.Getenv("PIN_TEST_AS_MAIN") != "" {
		main()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func pinCmd(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = append(os.Environ(), "PIN_TEST_AS_MAIN=1")
	var out, errOut strings.Builder
	cmd.Stdout, cmd.Stderr = &out, &errOut
	err := cmd.Run()
	if exit, ok := err.(*exec.ExitError); ok {
		code = exit.ExitCode()
	} else if err != nil {
		t.Fatalf("pin %v: %v", args, err)
	}
	return out.String(), errOut.String(), code
}

// The Makefile and the release workflow loop over `pin platforms` line by line: it must
// print exactly the pinned pairs, one per line, and nothing else.
func TestPlatformsPrintsOnePinnedPairPerLine(t *testing.T) {
	stdout, stderr, code := pinCmd(t, "platforms")
	want := strings.Join(pin.Platforms(), "\n") + "\n"
	if code != 0 || stdout != want || stderr != "" {
		t.Fatalf("pin platforms: exit %d, stdout %q, stderr %q; want exit 0, stdout %q", code, stdout, stderr, want)
	}
}

// A release for a platform ff does not ship stops before any download, naming the ones it does.
func TestFetchForAnUnshippedPlatformFailsNamingTheShippedOnes(t *testing.T) {
	_, stderr, code := pinCmd(t, "fetch", "windows", "amd64")
	if code != 1 || !strings.Contains(stderr, "linux/amd64") {
		t.Fatalf("pin fetch windows amd64: exit %d, stderr %q; want exit 1 naming linux/amd64", code, stderr)
	}
}

// A mistyped invocation in a build script fails loudly with the usage, not silently.
func TestAnUnknownCommandFailsWithTheUsage(t *testing.T) {
	_, stderr, code := pinCmd(t, "plattforms")
	if code != 1 || !strings.Contains(stderr, "usage: pin platforms | pin fetch") {
		t.Fatalf("pin plattforms: exit %d, stderr %q; want exit 1 with the usage", code, stderr)
	}
}
