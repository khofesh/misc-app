package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMainConvertsFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "input"), 0755); err != nil {
		t.Fatal(err)
	}
	html := `<html><body><h1>Title</h1><ul><li>Item</li></ul></body></html>`
	if err := os.WriteFile(filepath.Join(dir, "input", "input.html"), []byte(html), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	// output/ does not exist yet: main should create it
	main()

	got, err := os.ReadFile(filepath.Join(dir, "output", "html2text-result.txt"))
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	for _, want := range []string{"TITLE", "• Item"} {
		if !strings.Contains(string(got), want) {
			t.Errorf("output %q does not contain %q", got, want)
		}
	}
}
