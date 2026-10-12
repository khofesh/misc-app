package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// runMain re-runs this test binary as the decode-encode program inside dir.
func runMain(t *testing.T, dir string, args ...string) (string, int) {
	t.Helper()

	cmd := exec.Command(os.Args[0], append([]string{"-test.run=TestMainProcess", "--"}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "DECODE_ENCODE_RUN_MAIN=1")
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
	if os.Getenv("DECODE_ENCODE_RUN_MAIN") != "1" {
		return
	}
	for i, a := range os.Args {
		if a == "--" {
			os.Args = append([]string{"decode-encode"}, os.Args[i+1:]...)
			break
		}
	}
	main()
	os.Exit(0)
}

// setupDir creates input/ and output/ in a temp dir and writes the input file.
func setupDir(t *testing.T, inputName, content string) string {
	t.Helper()

	dir := t.TempDir()
	for _, sub := range []string{"input", "output"} {
		if err := os.Mkdir(filepath.Join(dir, sub), 0755); err != nil {
			t.Fatal(err)
		}
	}
	if inputName != "" {
		if err := os.WriteFile(filepath.Join(dir, "input", inputName), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func readOutput(t *testing.T, dir, name string) string {
	t.Helper()

	b, err := os.ReadFile(filepath.Join(dir, "output", name))
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	return string(b)
}

func TestEncode(t *testing.T) {
	dir := setupDir(t, "encode.txt", "a b&c=d/é")

	out, code := runMain(t, dir, "-encode")
	if code != 0 {
		t.Fatalf("exit code = %d, output:\n%s", code, out)
	}

	if got, want := readOutput(t, dir, "encoded.txt"), "a+b%26c%3Dd%2F%C3%A9"; got != want {
		t.Errorf("encoded = %q, want %q", got, want)
	}
}

func TestDecode(t *testing.T) {
	dir := setupDir(t, "decode.txt", "a+b%26c%3Dd%2F%C3%A9")

	out, code := runMain(t, dir, "-decode")
	if code != 0 {
		t.Fatalf("exit code = %d, output:\n%s", code, out)
	}

	if got, want := readOutput(t, dir, "decoded.txt"), "a b&c=d/é"; got != want {
		t.Errorf("decoded = %q, want %q", got, want)
	}
}

func TestDecodeInvalidEscape(t *testing.T) {
	dir := setupDir(t, "decode.txt", "bad%zz")

	if out, code := runMain(t, dir, "-decode"); code != 1 {
		t.Errorf("exit code = %d, want 1, output:\n%s", code, out)
	}
}

func TestMissingInputFile(t *testing.T) {
	dir := setupDir(t, "", "")

	if out, code := runMain(t, dir, "-encode"); code != 1 {
		t.Errorf("exit code = %d, want 1, output:\n%s", code, out)
	}
}

func TestFlagValidation(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"no flags", nil},
		{"both flags", []string{"-encode", "-decode"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupDir(t, "encode.txt", "x")

			if out, code := runMain(t, dir, tt.args...); code != 1 {
				t.Errorf("exit code = %d, want 1, output:\n%s", code, out)
			}
			if _, err := os.Stat(filepath.Join(dir, "output", "encoded.txt")); err == nil {
				t.Error("output file was written despite invalid flags")
			}
		})
	}
}
