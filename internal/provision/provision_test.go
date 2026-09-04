package provision

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// A stand-in runtime archive: the pinned layout with a fake binary in it.
func fakeArchive(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	zipped := gzip.NewWriter(&buffer)
	archive := tar.NewWriter(zipped)
	binary := []byte("#!/bin/sh\necho fake\n")
	for _, step := range []func() error{
		func() error {
			return archive.WriteHeader(&tar.Header{
				Name: runtimeDir + "/", Typeflag: tar.TypeDir, Mode: 0o755})
		},
		func() error {
			if err := archive.WriteHeader(&tar.Header{
				Name: runtimeDir + "/" + runtimeBin, Typeflag: tar.TypeReg,
				Mode: 0o755, Size: int64(len(binary))}); err != nil {
				return err
			}
			_, err := archive.Write(binary)
			return err
		},
		func() error {
			return archive.WriteHeader(&tar.Header{
				Name: runtimeDir + "/libggml.dylib", Typeflag: tar.TypeSymlink,
				Linkname: "libggml.0.dylib"})
		},
		archive.Close,
		zipped.Close,
	} {
		if err := step(); err != nil {
			t.Fatal(err)
		}
	}
	return buffer.Bytes()
}

// servePinned swaps the package's pinned artifact for a served fake, restoring it
// after. (Not called pin: that name is the pins' own package.)
func servePinned(t *testing.T, content []byte) *int {
	t.Helper()
	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		_, _ = w.Write(content)
	}))
	t.Cleanup(server.Close)

	sum := sha256.Sum256(content)
	saved := runtimeArchive
	runtimeArchive = artifact{url: server.URL, sha256: hex.EncodeToString(sum[:]), name: saved.name}
	t.Cleanup(func() { runtimeArchive = saved })
	return &hits
}

func TestRuntimeDownloadsVerifiesExtractsOnce(t *testing.T) {
	content := fakeArchive(t)
	hits := servePinned(t, content)
	cache := t.TempDir()

	bin, err := Runtime(cache)
	if err != nil {
		t.Fatalf("runtime: %v", err)
	}
	info, err := os.Stat(bin)
	if err != nil {
		t.Fatalf("binary missing: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatal("the exec bit did not survive extraction")
	}
	if _, err := os.Readlink(filepath.Join(cache, "runtime", runtimeDir, "libggml.dylib")); err != nil {
		t.Fatalf("the dylib symlink did not survive: %v", err)
	}

	// The second ask is answered from the cache, not the network.
	if _, err := Runtime(cache); err != nil {
		t.Fatalf("second runtime: %v", err)
	}
	if *hits != 1 {
		t.Fatalf("server was hit %d times, want 1", *hits)
	}
}

func TestACorruptDownloadIsRefusedWholesale(t *testing.T) {
	content := fakeArchive(t)
	servePinned(t, content)
	// Pin a hash the served bytes cannot match.
	runtimeArchive.sha256 = "0000000000000000000000000000000000000000000000000000000000000000"

	cache := t.TempDir()
	if _, err := Runtime(cache); err == nil {
		t.Fatal("a checksum mismatch was accepted")
	}
	if _, err := os.Stat(filepath.Join(cache, runtimeArchive.name)); err == nil {
		t.Fatal("a corrupt artifact was left in place under its final name")
	}
}

func TestAnEscapingArchiveEntryIsRefused(t *testing.T) {
	var buffer bytes.Buffer
	zipped := gzip.NewWriter(&buffer)
	archive := tar.NewWriter(zipped)
	payload := []byte("nope")
	if err := archive.WriteHeader(&tar.Header{
		Name: "../escape", Typeflag: tar.TypeReg, Mode: 0o644,
		Size: int64(len(payload))}); err != nil {
		t.Fatal(err)
	}
	_, _ = archive.Write(payload)
	_ = archive.Close()
	_ = zipped.Close()

	if err := untar(&buffer, filepath.Join(t.TempDir(), "out")); err == nil {
		t.Fatal("an entry escaping the destination was extracted")
	}
}

// The bundled-runtime path: the blob extracts when it matches the pin and is refused
// wholesale when it does not — a build that bundled the wrong bytes must not run them.
func TestABundledRuntimeIsVerifiedBeforeExtraction(t *testing.T) {
	blob := fakeArchive(t)
	sum := sha256.Sum256(blob)

	saved := runtimeArchive
	runtimeArchive.sha256 = hex.EncodeToString(sum[:])
	t.Cleanup(func() { runtimeArchive = saved })

	cache := t.TempDir()
	bin, err := runtimeFromBlob(cache, blob)
	if err != nil {
		t.Fatalf("runtimeFromBlob: %v", err)
	}
	if info, err := os.Stat(bin); err != nil || info.Mode()&0o111 == 0 {
		t.Fatalf("extracted binary bad: %v", err)
	}

	runtimeArchive.sha256 = "0000000000000000000000000000000000000000000000000000000000000000"
	if _, err := runtimeFromBlob(t.TempDir(), blob); err == nil {
		t.Fatal("a mismatched bundled runtime was extracted")
	}
}

func TestFFModelFileShortCircuitsProvisioning(t *testing.T) {
	own := filepath.Join(t.TempDir(), "own.gguf")
	if err := os.WriteFile(own, []byte("gguf"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FF_MODEL_FILE", own)
	got, err := Model(t.TempDir())
	if err != nil || got != own {
		t.Fatalf("model = %q, %v; want the user's own file", got, err)
	}

	t.Setenv("FF_MODEL_FILE", filepath.Join(t.TempDir(), "missing.gguf"))
	if _, err := Model(t.TempDir()); err == nil {
		t.Fatal("a missing FF_MODEL_FILE was accepted")
	}
}
