package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runMain re-runs this test binary as the pdf2md program inside dir.
func runMain(t *testing.T, dir string, args ...string) (string, string, int) {
	t.Helper()

	cmd := exec.Command(os.Args[0], append([]string{"-test.run=TestMainProcess", "--"}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PDF2MD_RUN_MAIN=1")
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()

	if exitErr, ok := err.(*exec.ExitError); ok {
		return stdout.String(), stderr.String(), exitErr.ExitCode()
	}
	if err != nil {
		t.Fatalf("running main: %v", err)
	}
	return stdout.String(), stderr.String(), 0
}

// TestMainProcess is not a real test: it runs main() when started by runMain.
func TestMainProcess(t *testing.T) {
	if os.Getenv("PDF2MD_RUN_MAIN") != "1" {
		return
	}
	for i, a := range os.Args {
		if a == "--" {
			os.Args = append([]string{"pdf2md"}, os.Args[i+1:]...)
			break
		}
	}
	main()
	os.Exit(0)
}

// setupDir creates a temp dir with input/ holding the given fixture PDFs.
func setupDir(t *testing.T, fixtures ...string) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "input"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range fixtures {
		b, err := os.ReadFile(filepath.Join("..", "..", "internal", "pdf2md", "testdata", f))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "input", f), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestExitCodes(t *testing.T) {
	dir := setupDir(t, "basic.pdf")
	tests := []struct {
		name string
		args []string
		code int
	}{
		{"missing -i", nil, 1},
		{"bad -ocr", []string{"-i", "input/basic.pdf", "-ocr", "maybe"}, 1},
		{"bad -backend", []string{"-i", "input/basic.pdf", "-backend", "x"}, 1},
		{"bad -pages", []string{"-i", "input/basic.pdf", "-pages", "z"}, 1},
		{"missing input", []string{"-i", "input/none.pdf"}, 1},
		{"images to stdout", []string{"-i", "input/basic.pdf", "-images"}, 1},
		{"directory to stdout", []string{"-i", "input"}, 1},
		{"missing mutool", []string{"-i", "input/basic.pdf", "-backend", "mutool", "-mutool", "/nonexistent/mutool"}, 2},
		{"help", []string{"-h"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, stderr, code := runMain(t, dir, tt.args...)
			if code != tt.code {
				t.Errorf("exit code = %d, want %d, stderr:\n%s", code, tt.code, stderr)
			}
		})
	}
}

func TestStdout(t *testing.T) {
	dir := setupDir(t, "basic.pdf")
	stdout, stderr, code := runMain(t, dir, "-i", "input/basic.pdf", "-backend", "native")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr:\n%s", code, stderr)
	}
	if !strings.HasPrefix(stdout, "# Quarterly Report\n") {
		t.Errorf("stdout = %q", stdout)
	}
	if strings.Contains(stdout, "INF") {
		t.Error("logs written to stdout")
	}
}

func TestOutputFile(t *testing.T) {
	dir := setupDir(t, "basic.pdf")
	out := filepath.Join(dir, "output", "basic.md")

	if _, stderr, code := runMain(t, dir, "-i", "input/basic.pdf", "-o", "output/basic.md", "-backend", "native"); code != 0 {
		t.Fatalf("exit code = %d, stderr:\n%s", code, stderr)
	}
	b, err := os.ReadFile(out)
	if err != nil || !strings.Contains(string(b), "## Highlights") {
		t.Fatalf("output = %q, err %v", b, err)
	}

	if err := os.WriteFile(out, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, stderr, code := runMain(t, dir, "-i", "input/basic.pdf", "-o", "output/basic.md", "-backend", "native")
	if b, _ := os.ReadFile(out); code != 0 || string(b) != "keep" || !strings.Contains(stderr, "output exists") {
		t.Errorf("existing output: code %d, content %q, stderr:\n%s", code, b, stderr)
	}

	runMain(t, dir, "-i", "input/basic.pdf", "-o", "output/basic.md", "-backend", "native", "-overwrite")
	if b, _ := os.ReadFile(out); string(b) == "keep" {
		t.Error("-overwrite did not replace the output")
	}
}

func TestDirectory(t *testing.T) {
	dir := setupDir(t, "basic.pdf", "table.pdf")
	if err := os.WriteFile(filepath.Join(dir, "input", "broken.pdf"), []byte("not a pdf"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "input", "notes.txt"), []byte("skip me"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr, code := runMain(t, dir, "-i", "input", "-o", "output", "-backend", "native")
	if code != 3 {
		t.Errorf("exit code = %d, want 3, stderr:\n%s", code, stderr)
	}
	if !strings.Contains(stderr, "failed=1") || !strings.Contains(stderr, "ok=2") {
		t.Errorf("missing summary in stderr:\n%s", stderr)
	}
	for _, name := range []string{"basic.md", "table.md"} {
		if _, err := os.Stat(filepath.Join(dir, "output", name)); err != nil {
			t.Errorf("%s not written: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "output", "notes.md")); err == nil {
		t.Error("non-PDF file converted")
	}
}

func TestImages(t *testing.T) {
	dir := setupDir(t, "scan.pdf")
	_, stderr, code := runMain(t, dir, "-i", "input/scan.pdf", "-o", "output/scan.md", "-images", "-ocr", "never", "-backend", "native")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr:\n%s", code, stderr)
	}
	b, err := os.ReadFile(filepath.Join(dir, "output", "scan.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "![](scan_images/p1_1.png)") {
		t.Errorf("image link missing:\n%s", b)
	}
	if _, err := os.Stat(filepath.Join(dir, "output", "scan_images", "p1_1.png")); err != nil {
		t.Errorf("image not written: %v", err)
	}
}
