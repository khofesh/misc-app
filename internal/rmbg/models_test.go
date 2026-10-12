package rmbg

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLookup(t *testing.T) {
	if _, ok := Lookup("u2net"); !ok {
		t.Error("u2net missing from registry")
	}
	if _, ok := Lookup("nope"); ok {
		t.Error("unknown model found in registry")
	}
	if len(RegistryNames()) != len(registry) {
		t.Errorf("RegistryNames returned %d names, want %d", len(RegistryNames()), len(registry))
	}
}

func TestRegistryChecksumsPinned(t *testing.T) {
	for name, spec := range registry {
		if len(spec.SHA256) != 64 {
			t.Errorf("%s: sha256 %q is not 64 hex chars", name, spec.SHA256)
		}
		if spec.URL == "" || spec.Size <= 0 {
			t.Errorf("%s: incomplete spec %+v", name, spec)
		}
	}
}

func TestIsModelPath(t *testing.T) {
	dir := t.TempDir()
	model := filepath.Join(dir, "model.onnx")
	if err := os.WriteFile(model, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !isModelPath(model) {
		t.Error("existing .onnx file not recognized as a path")
	}
	if !isModelPath("does-not-exist.onnx") {
		t.Error(".onnx suffix not recognized as a path")
	}
	if isModelPath("u2net") {
		t.Error("registry name wrongly recognized as a path")
	}
}

func TestModelPathCustom(t *testing.T) {
	dir := t.TempDir()
	model := filepath.Join(dir, "custom.onnx")
	if err := os.WriteFile(model, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	path, spec, err := ModelPath(model)
	if err != nil {
		t.Fatal(err)
	}
	if path != model {
		t.Errorf("path = %q, want %q", path, model)
	}
	if spec.Name != "custom.onnx" {
		t.Errorf("spec.Name = %q, want custom.onnx", spec.Name)
	}
}

func TestModelPathMissingCustom(t *testing.T) {
	_, _, err := ModelPath("missing.onnx")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !errors.Is(err, ErrInvalidModel) {
		t.Errorf("error %v is not ErrInvalidModel", err)
	}
}

func TestModelPathUnknown(t *testing.T) {
	_, _, err := ModelPath("not-a-model")
	if !errors.Is(err, ErrInvalidModel) {
		t.Errorf("error %v is not ErrInvalidModel", err)
	}
}

func TestVerifyChecksum(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	// sha256("hello")
	const sum = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"

	if err := verifyChecksum(path, sum); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := verifyChecksum(path, "deadbeef"); err == nil {
		t.Error("expected checksum mismatch")
	}
}

func TestCacheDir(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "/tmp/xdg")
	dir, err := CacheDir()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join("/tmp/xdg", "rmbg"); dir != want {
		t.Errorf("CacheDir = %q, want %q", dir, want)
	}
}
