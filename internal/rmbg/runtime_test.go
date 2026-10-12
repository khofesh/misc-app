//go:build linux

package rmbg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindORTLibExplicitMissing(t *testing.T) {
	if _, err := FindORTLib(filepath.Join(t.TempDir(), "nope.so")); err == nil {
		t.Fatal("expected an error for a missing -ort-lib path")
	}
}

func TestFindORTLibExplicit(t *testing.T) {
	dir := t.TempDir()
	lib := filepath.Join(dir, "libonnxruntime.so")
	if err := os.WriteFile(lib, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := FindORTLib(lib)
	if err != nil {
		t.Fatal(err)
	}
	if got != lib {
		t.Errorf("got %q, want %q", got, lib)
	}
}

func TestFindORTLibWorkingDir(t *testing.T) {
	t.Setenv("ONNXRUNTIME_LIB", "")
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "lib", "libonnxruntime.so"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	got, err := FindORTLib("")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join("lib", "libonnxruntime.so"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFindORTLibEnv(t *testing.T) {
	t.Setenv("ONNXRUNTIME_LIB", "/custom/libonnxruntime.so")
	got, err := FindORTLib("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/custom/libonnxruntime.so" {
		t.Errorf("got %q, want the environment value", got)
	}
}
