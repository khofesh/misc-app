//go:build linux

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// runMain re-runs this test binary as the rmbg program inside dir.
func runMain(t *testing.T, dir string, args ...string) (string, int) {
	t.Helper()

	cmd := exec.Command(os.Args[0], append([]string{"-test.run=TestMainProcess", "--"}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "RMBG_RUN_MAIN=1")
	out, err := cmd.CombinedOutput()

	if exitErr, ok := err.(*exec.ExitError); ok {
		return string(out), exitErr.ExitCode()
	}
	if err != nil {
		t.Fatalf("running main: %v", err)
	}
	return string(out), 0
}

// TestMainProcess is not a real test: it runs main() when started by runMain.
func TestMainProcess(t *testing.T) {
	if os.Getenv("RMBG_RUN_MAIN") != "1" {
		return
	}
	for i, a := range os.Args {
		if a == "--" {
			os.Args = append([]string{"rmbg"}, os.Args[i+1:]...)
			break
		}
	}
	main()
	os.Exit(0)
}

func TestFlagErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"no flags", nil},
		{"missing output", []string{"-i", "input.png"}},
		{"invalid device", []string{"-i", "input.png", "-o", "out.png", "-device", "tpu"}},
		{"invalid threshold", []string{"-i", "input.png", "-o", "out.png", "-alpha-threshold", "300"}},
		{"invalid bg", []string{"-i", "input.png", "-o", "out.png", "-bg", "nope"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "input.png"), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
			if out, code := runMain(t, dir, tt.args...); code != 1 {
				t.Errorf("exit code = %d, want 1, output:\n%s", code, out)
			}
		})
	}
}

func TestMissingInput(t *testing.T) {
	dir := t.TempDir()
	out, code := runMain(t, dir, "-i", "nope.png", "-o", "out.png")
	if code != 1 {
		t.Errorf("exit code = %d, want 1, output:\n%s", code, out)
	}
}

func TestUnknownModel(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "input.png"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// An unknown model is an input error (exit 1), caught before ONNX Runtime
	// is loaded, so this does not need a model or a shared library.
	out, code := runMain(t, dir, "-i", "input.png", "-o", "out.png", "-model", "not-a-model")
	if code != 1 {
		t.Errorf("exit code = %d, want 1, output:\n%s", code, out)
	}
}

func TestOutputBase(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"photo.jpg", "photo"},
		{"photo.JPEG", "photo"},
		{"a.b.png", "a.b"},
		{"noext", "noext"},
	}
	for _, tt := range tests {
		if got := outputBase(tt.in); got != tt.want {
			t.Errorf("outputBase(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestIsSupportedImage(t *testing.T) {
	for _, name := range []string{"a.jpg", "a.JPEG", "a.png", "a.webp"} {
		if !isSupportedImage(name) {
			t.Errorf("%q should be supported", name)
		}
	}
	for _, name := range []string{"a.gif", "a.txt", "a"} {
		if isSupportedImage(name) {
			t.Errorf("%q should not be supported", name)
		}
	}
}

func TestIsJPEG(t *testing.T) {
	if !isJPEG("a.jpg") || !isJPEG("a.jpeg") {
		t.Error("jpg/jpeg not recognized")
	}
	if isJPEG("a.png") {
		t.Error("png wrongly recognized as jpeg")
	}
}
