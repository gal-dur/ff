// Package provision fetches the pinned runtime and model into the cache, once.
//
// Every artifact is named by URL and sha256 in this file — the one place a version
// bump or a second platform would be added (see SPEC.md's pinning table). Downloads
// land as .partial, are verified, and only then renamed into place: a torn download
// or a swapped upstream file cannot be used.
package provision

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// The pinned artifacts. The runtime build and the model move independently and each
// carries its own checksum.
var (
	runtimeArchive = artifact{
		url: "https://github.com/ggml-org/llama.cpp/releases/download/b10797/" +
			"llama-b10797-bin-macos-arm64.tar.gz",
		sha256: "474a788ec73d17a066360b1c50c9733c78a47d062616e91963c65a344548e889",
		name:   "llama-b10797-bin-macos-arm64.tar.gz",
	}
	// The archive's top-level directory, and the binary inside it.
	runtimeDir = "llama-b10797"
	runtimeBin = "llama-cli"

	modelFile = artifact{
		url: "https://huggingface.co/Qwen/Qwen2.5-Coder-3B-Instruct-GGUF/resolve/main/" +
			"qwen2.5-coder-3b-instruct-q4_k_m.gguf",
		sha256: "724fb256bec1ff062b2f65e4569e871ad2e95ab2a3989723d1769c54294730b7",
		name:   "qwen2.5-coder-3b-instruct-q4_k_m.gguf",
	}
)

type artifact struct {
	url, sha256, name string
}

// Runtime ensures llama-cli is in the cache and answers its path.
func Runtime(cache string) (string, error) {
	bin := filepath.Join(cache, "runtime", runtimeDir, runtimeBin)
	if _, err := os.Stat(bin); err == nil {
		return bin, nil
	}
	archive, err := ensure(cache, runtimeArchive)
	if err != nil {
		return "", err
	}
	// Extracted beside its dylibs — the binary loads them by relative rpath.
	if err := untar(archive, filepath.Join(cache, "runtime")); err != nil {
		return "", err
	}
	if _, err := os.Stat(bin); err != nil {
		return "", fmt.Errorf("archive did not contain %s/%s", runtimeDir, runtimeBin)
	}
	return bin, nil
}

// Model ensures the GGUF is in the cache and answers its path. FF_MODEL_FILE points
// at any other local model instead; a user-supplied file is never re-verified.
func Model(cache string) (string, error) {
	if own := os.Getenv("FF_MODEL_FILE"); own != "" {
		if _, err := os.Stat(own); err != nil {
			return "", fmt.Errorf("FF_MODEL_FILE: %w", err)
		}
		return own, nil
	}
	return ensure(cache, modelFile)
}

// ensure answers the verified artifact's path, downloading it first if needed.
func ensure(cache string, a artifact) (string, error) {
	final := filepath.Join(cache, a.name)
	if _, err := os.Stat(final); err == nil {
		return final, nil
	}
	if err := os.MkdirAll(cache, 0o755); err != nil {
		return "", err
	}

	partial := final + ".partial"
	if err := download(a.url, partial); err != nil {
		_ = os.Remove(partial)
		return "", err
	}
	sum, err := fileSHA256(partial)
	if err != nil {
		return "", err
	}
	if sum != a.sha256 {
		_ = os.Remove(partial)
		return "", fmt.Errorf("%s: checksum mismatch (got %s, pinned %s)", a.name, sum, a.sha256)
	}
	return final, os.Rename(partial, final)
}

func download(url, dest string) error {
	fmt.Fprintf(os.Stderr, "ff: fetching %s\n", url)
	client := &http.Client{Timeout: 2 * time.Hour}
	response, err := client.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", url, response.Status)
	}
	file, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := io.Copy(file, progress{total: response.ContentLength}.wrap(response.Body)); err != nil {
		return err
	}
	return file.Close()
}

// progress prints a line per ~10%, to stderr, only for downloads big enough to wait
// on. A 2 GB fetch with no output reads as a hang.
type progress struct{ total int64 }

func (p progress) wrap(r io.Reader) io.Reader {
	if p.total < 50<<20 {
		return r
	}
	return &progressReader{inner: r, total: p.total, step: p.total / 10}
}

type progressReader struct {
	inner       io.Reader
	total, done int64
	step, next  int64
}

func (r *progressReader) Read(buffer []byte) (int, error) {
	n, err := r.inner.Read(buffer)
	r.done += int64(n)
	if r.done >= r.next {
		fmt.Fprintf(os.Stderr, "ff: %d%%\n", r.done*100/r.total)
		r.next += r.step
	}
	return n, err
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

// untar extracts a .tar.gz, preserving the exec bit and refusing entries that would
// escape the destination.
func untar(archive, dest string) error {
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer file.Close()
	unzipped, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	reader := tar.NewReader(unzipped)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		target := filepath.Join(dest, filepath.Clean(header.Name))
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("archive entry escapes destination: %q", header.Name)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
				os.FileMode(header.Mode)&0o777)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, reader); err != nil {
				_ = out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		case tar.TypeSymlink:
			// The runtime archive links versioned dylibs to their sonames.
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			_ = os.Remove(target)
			if err := os.Symlink(header.Linkname, target); err != nil {
				return err
			}
		}
	}
}
