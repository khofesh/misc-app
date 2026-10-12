package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMainConvertsFile(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"input", "output"} {
		if err := os.Mkdir(filepath.Join(dir, sub), 0755); err != nil {
			t.Fatal(err)
		}
	}
	html := `<h1>Title</h1><p>Some <strong>bold</strong> text and a <a href="https://example.com">link</a>.</p><ul><li>One</li></ul>`
	if err := os.WriteFile(filepath.Join(dir, "input", "input.html"), []byte(html), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	main()

	got, err := os.ReadFile(filepath.Join(dir, "output", "output.md"))
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	for _, want := range []string{
		"# Title",
		"**bold**",
		"[link](https://example.com)",
		"- One",
	} {
		if !strings.Contains(string(got), want) {
			t.Errorf("output %q does not contain %q", got, want)
		}
	}
}
