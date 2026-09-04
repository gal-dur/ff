// Package provision fetches the pinned runtime and model into the cache, once.
//
// Every artifact is named by URL and sha256 in internal/pin — the one place a
// version bump or another platform would be added. Downloads land as .partial, are
// verified, and only then renamed into place: a torn download or a swapped upstream
// file cannot be used.
package provision

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gal-dur/ff/internal/pin"
)

// The pinned artifacts, named once in internal/pin. The runtime build and the model
// move independently and each carries its own checksum. Vars rather than consts so
// the tests can point them at a served fake. The runtime pin is the platform's own:
// a binary built for an unpinned platform reports that on first use, not at init.
var (
	runtimeArchive, runtimeErr = currentRuntime()
	runtimeDir                 = pin.RuntimeDir
	runtimeBin                 = "llama-cli"

	modelFile = artifact{url: pin.ModelURL, sha256: pin.ModelSHA256, name: pin.ModelFile}
)

func currentRuntime() (artifact, error) {
	r, err := pin.RuntimeFor(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return artifact{}, err
	}
	return artifact{url: r.URL, sha256: r.SHA256, name: r.Archive}, nil
}

type artifact struct {
	url, sha256, name string
}

// Runtime ensures llama-cli is in the cache and answers its path. A release build
// carries the archive inside the binary (see runtime_embed.go) and extracts it; a
// build without it downloads the same pinned archive instead.
func Runtime(cache string) (string, error) {
	if runtimeErr != nil {
		return "", runtimeErr
	}
	bin := filepath.Join(cache, "runtime", runtimeDir, runtimeBin)
	if _, err := os.Stat(bin); err == nil {
		return bin, nil
	}
	if len(embeddedRuntime) > 0 {
		return runtimeFromBlob(cache, embeddedRuntime)
	}
	archive, err := ensure(cache, runtimeArchive)
	if err != nil {
		return "", err
	}
	file, err := os.Open(archive)
	if err != nil {
		return "", err
	}
	defer file.Close()
	// Extracted beside its dylibs — the binary loads them by relative rpath.
	return runtimeBinAfter(cache, untar(file, filepath.Join(cache, "runtime")))
}

// runtimeFromBlob extracts the bundled archive, refusing bytes that do not match the
// pin — a build that bundled the wrong file must not run it.
func runtimeFromBlob(cache string, blob []byte) (string, error) {
	sum := sha256.Sum256(blob)
	if hex.EncodeToString(sum[:]) != runtimeArchive.sha256 {
		return "", fmt.Errorf("the bundled runtime does not match the pin; rebuild")
	}
	return runtimeBinAfter(cache, untar(bytes.NewReader(blob), filepath.Join(cache, "runtime")))
}

func runtimeBinAfter(cache string, err error) (string, error) {
	if err != nil {
		return "", err
	}
	bin := filepath.Join(cache, "runtime", runtimeDir, runtimeBin)
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

	// A failed download keeps its .partial: the next run resumes it with a Range
	// request instead of starting a multi-gigabyte fetch over.
	partial := final + ".partial"
	if err := download(a.url, partial); err != nil {
		return "", fmt.Errorf("%w (partial download kept; rerun to resume)", err)
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

// download fetches url into dest, resuming whatever dest already holds. A dropped
// connection retries in-process a couple of times, each time picking up where the
// bytes stopped; both hosts ff pins (GitHub, Hugging Face) honor Range requests.
func download(url, dest string) error {
	fmt.Fprintf(os.Stderr, "ff: fetching %s\n", url)
	client := &http.Client{Timeout: 2 * time.Hour}
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			fmt.Fprintf(os.Stderr, "ff: download interrupted (%v); resuming\n", err)
			time.Sleep(2 * time.Second)
		}
		if err = fetch(client, url, dest); err == nil {
			return nil
		}
	}
	return err
}

func fetch(client *http.Client, url, dest string) error {
	var offset int64
	if info, err := os.Stat(dest); err == nil {
		offset = info.Size()
	}
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if offset > 0 {
		request.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	switch {
	case offset > 0 && response.StatusCode == http.StatusPartialContent:
		// The server honors the resume; append to what is there.
	case response.StatusCode == http.StatusOK:
		// A full body — the server ignored the Range, or there was none. Start over.
		offset = 0
	default:
		return fmt.Errorf("GET %s: %s", url, response.Status)
	}

	flags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	if offset > 0 {
		flags = os.O_CREATE | os.O_WRONLY | os.O_APPEND
	}
	file, err := os.OpenFile(dest, flags, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	total := response.ContentLength
	if total > 0 {
		total += offset
	}
	reader := progress{total: total, done: offset}.wrap(response.Body)
	if _, err := io.Copy(file, reader); err != nil {
		return err
	}
	return file.Close()
}

// progress prints a line per ~10%, to stderr, only for downloads big enough to wait
// on. A 2 GB fetch with no output reads as a hang. done carries a resumed fetch's
// starting offset, so the percentages stay honest across interruptions.
type progress struct{ total, done int64 }

func (p progress) wrap(r io.Reader) io.Reader {
	if p.total < 50<<20 {
		return r
	}
	step := p.total / 10
	next := (p.done/step + 1) * step
	return &progressReader{inner: r, total: p.total, done: p.done, step: step, next: next}
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

// untar extracts a .tar.gz stream, preserving the exec bit and refusing entries that
// would escape the destination.
func untar(archive io.Reader, dest string) error {
	unzipped, err := gzip.NewReader(archive)
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
