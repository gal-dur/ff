package pin

import (
	"slices"
	"strings"
	"testing"
)

// The Makefile and the release workflow loop over `pin platforms`: a roster out of order,
// or missing a pair, would ship a different set of binaries than the map pins.
func TestPlatformsListEveryPinnedPairSorted(t *testing.T) {
	want := []string{"darwin/amd64", "darwin/arm64", "linux/amd64", "linux/arm64"}
	if got := Platforms(); !slices.Equal(got, want) {
		t.Errorf("Platforms() = %v, want %v", got, want)
	}
}

// A build for a platform ff does not ship says which ones it does, so the fix is in
// the message.
func TestRuntimeForAnUnshippedPlatformNamesTheShippedOnes(t *testing.T) {
	_, err := RuntimeFor("windows", "amd64")
	if err == nil {
		t.Fatal("RuntimeFor(windows, amd64) answered a runtime")
	}
	for _, platform := range Platforms() {
		if !strings.Contains(err.Error(), platform) {
			t.Errorf("error %q does not name %s", err, platform)
		}
	}
}

// Every pinned runtime is the release's own archive for its platform: the URL is the
// release's download of the file the cache names it by, and the checksum a full sha256.
func TestEveryPinnedRuntimeIsItsReleaseArchive(t *testing.T) {
	for _, platform := range Platforms() {
		goos, goarch, _ := strings.Cut(platform, "/")
		a, err := RuntimeFor(goos, goarch)
		if err != nil {
			t.Fatal(err)
		}
		if a.URL != runtimeBase+a.Name || !strings.HasPrefix(a.Name, RuntimeDir+"-bin-") {
			t.Errorf("%s: URL %s, name %s", platform, a.URL, a.Name)
		}
		if len(a.SHA256) != 64 || strings.Trim(a.SHA256, "0123456789abcdef") != "" {
			t.Errorf("%s: checksum %q is not a sha256", platform, a.SHA256)
		}
	}
}
