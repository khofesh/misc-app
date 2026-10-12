// Package rmbg removes the background from images using a local ONNX
// segmentation model through ONNX Runtime.
package rmbg

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ErrInvalidModel marks the user requesting a model that does not exist, as
// opposed to a model that exists but fails to load.
var ErrInvalidModel = errors.New("invalid model")

const modelReleaseBase = "https://github.com/danielgatis/rembg/releases/download/v0.0.0/"

var (
	imagenetMean  = [3]float32{0.485, 0.456, 0.406}
	imagenetStd   = [3]float32{0.229, 0.224, 0.225}
	inceptionMean = [3]float32{0.5, 0.5, 0.5}
	inceptionStd  = [3]float32{1.0, 1.0, 1.0}
)

// ModelSpec describes a segmentation model: where to fetch it, how to verify
// it, and how to feed it.
type ModelSpec struct {
	Name       string
	URL        string
	SHA256     string
	Size       int // square input side length; 0 means read it from the model
	Mean       [3]float32
	Std        [3]float32
	InputName  string
	OutputName string
}

// registry holds the built-in models. Populate it with Lookup.
var registry = map[string]ModelSpec{
	"u2net": {
		Name:       "u2net",
		URL:        modelReleaseBase + "u2net.onnx",
		SHA256:     "8d10d2f3bb75ae3b6d527c77944fc5e7dcd94b29809d47a739a7a728a912b491",
		Size:       320,
		Mean:       imagenetMean,
		Std:        imagenetStd,
		InputName:  "input.1",
		OutputName: "1959",
	},
	"u2netp": {
		Name:       "u2netp",
		URL:        modelReleaseBase + "u2netp.onnx",
		SHA256:     "309c8469258dda742793dce0ebea8e6dd393174f89934733ecc8b14c76f4ddd8",
		Size:       320,
		Mean:       imagenetMean,
		Std:        imagenetStd,
		InputName:  "input.1",
		OutputName: "1959",
	},
	"u2net_human_seg": {
		Name:       "u2net_human_seg",
		URL:        modelReleaseBase + "u2net_human_seg.onnx",
		SHA256:     "01eb6a29a5c4d8edb30b56adad9bb3a2a0535338e480724a213e0acfd2d1c73c",
		Size:       320,
		Mean:       imagenetMean,
		Std:        imagenetStd,
		InputName:  "input.1",
		OutputName: "1959",
	},
	"isnet-general-use": {
		Name:       "isnet-general-use",
		URL:        modelReleaseBase + "isnet-general-use.onnx",
		SHA256:     "60920e99c45464f2ba57bee2ad08c919a52bbf852739e96947fbb4358c0d964a",
		Size:       1024,
		Mean:       inceptionMean,
		Std:        inceptionStd,
		InputName:  "input.1",
		OutputName: "1955",
	},
}

// Lookup returns the registry entry for name.
func Lookup(name string) (ModelSpec, bool) {
	s, ok := registry[name]
	return s, ok
}

// RegistryNames returns the built-in model names.
func RegistryNames() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	return names
}

// customSpec is used when the user points -model at a .onnx file: its input
// size and tensor names are read from the model itself.
func customSpec(path string) ModelSpec {
	return ModelSpec{
		Name: filepath.Base(path),
		Mean: imagenetMean,
		Std:  imagenetStd,
	}
}

// isModelPath reports whether name refers to a local .onnx file rather than a
// registry entry.
func isModelPath(name string) bool {
	if strings.HasSuffix(strings.ToLower(name), ".onnx") {
		return true
	}
	info, err := os.Stat(name)
	return err == nil && !info.IsDir()
}

// CacheDir returns the model cache directory ($XDG_CACHE_HOME/rmbg, defaulting
// to ~/.cache/rmbg).
func CacheDir() (string, error) {
	base := os.Getenv("XDG_CACHE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("locating home directory: %w", err)
		}
		base = filepath.Join(home, ".cache")
	}
	return filepath.Join(base, "rmbg"), nil
}

// ModelPath resolves -model to a local .onnx path. Built-in names are
// downloaded and verified on first use; a path is used as-is.
func ModelPath(name string) (path string, spec ModelSpec, err error) {
	if isModelPath(name) {
		if _, err := os.Stat(name); err != nil {
			return "", ModelSpec{}, fmt.Errorf("%w: model file %q: %v", ErrInvalidModel, name, err)
		}
		return name, customSpec(name), nil
	}

	spec, ok := Lookup(name)
	if !ok {
		return "", ModelSpec{}, fmt.Errorf("%w: unknown model %q (known: %s)", ErrInvalidModel, name, strings.Join(RegistryNames(), ", "))
	}

	dir, err := CacheDir()
	if err != nil {
		return "", ModelSpec{}, err
	}
	path = filepath.Join(dir, spec.Name+".onnx")
	if err := ensureModel(path, spec, os.Stderr); err != nil {
		return "", ModelSpec{}, err
	}
	return path, spec, nil
}

// ensureModel makes sure a verified copy of the model exists at path,
// downloading it if needed.
func ensureModel(path string, spec ModelSpec, progress io.Writer) error {
	if _, err := os.Stat(path); err == nil {
		if err := verifyChecksum(path, spec.SHA256); err == nil {
			return nil
		}
		// A cached file with the wrong hash is corrupt; fetch it again.
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("removing corrupt model %q: %w", path, err)
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}
	fmt.Fprintf(progress, "downloading %s\n", spec.Name)

	tmp := path + ".part"
	defer os.Remove(tmp)
	if err := download(spec.URL, tmp); err != nil {
		return fmt.Errorf("downloading %s: %w", spec.Name, err)
	}
	if err := verifyChecksum(tmp, spec.SHA256); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("saving model: %w", err)
	}
	return nil
}

// download fetches url into dst, printing progress to stderr.
func download(url, dst string) error {
	resp, err := http.Get(url) //nolint:gosec // URL comes from the pinned registry
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s", resp.Status)
	}

	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()

	reporter := &progressWriter{total: resp.ContentLength, w: os.Stderr}
	if _, err := io.Copy(io.MultiWriter(f, reporter), resp.Body); err != nil {
		return err
	}
	reporter.done()
	return f.Close()
}

// progressWriter prints a percentage to w as bytes are written.
type progressWriter struct {
	total   int64
	written int64
	last    int
	w       io.Writer
}

func (p *progressWriter) Write(b []byte) (int, error) {
	n := len(b)
	p.written += int64(n)
	if p.total > 0 {
		pc := int(p.written * 100 / p.total)
		if pc != p.last {
			p.last = pc
			fmt.Fprintf(p.w, "\r  %3d%%", pc)
		}
	}
	return n, nil
}

func (p *progressWriter) done() {
	if p.total > 0 {
		fmt.Fprintln(p.w)
	}
}

// verifyChecksum returns an error if the SHA-256 of path does not match want.
func verifyChecksum(path, want string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening %q: %w", path, err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hashing %q: %w", path, err)
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != want {
		return fmt.Errorf("checksum mismatch for %q: got %s, want %s", path, got, want)
	}
	return nil
}
