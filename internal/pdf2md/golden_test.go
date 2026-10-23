package pdf2md

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite testdata/*.golden.md")

var fixtures = []string{"basic", "twocol", "table", "code", "link"}

func convertFixture(t *testing.T, name string, opts Options) string {
	t.Helper()
	c, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	if _, err := c.Convert(context.Background(), filepath.Join("testdata", name+".pdf"), &b); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func TestGolden(t *testing.T) {
	if FindMutool("") == "" {
		t.Skip("mutool not installed")
	}
	for _, name := range fixtures {
		t.Run(name, func(t *testing.T) {
			got := convertFixture(t, name, Options{Backend: BackendMutool, OCR: OCRNever, Tables: true})
			path := filepath.Join("testdata", name+".golden.md")
			if *update {
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Errorf("output differs from %s (run with -update to accept)\ngot:\n%s\nwant:\n%s", path, got, want)
			}
		})
	}
}

func TestNativeBackend(t *testing.T) {
	tests := map[string][]string{
		"basic":  {"# Quarterly Report", "## Highlights", "- Costs stayed flat", "1. Hire two engineers", "**ambitious**", "visible before the end"},
		"twocol": {"# A Study of Two Columns", "first.\n\nA second paragraph", "left column.\n\nRight column"},
		"table":  {"| Coffee beans | kg | 24.50 |"},
		"code":   {"```\nfunc main() {"},
		"link":   {"[MuPDF website](https://mupdf.com)"},
	}
	for _, name := range fixtures {
		t.Run(name, func(t *testing.T) {
			got := convertFixture(t, name, Options{Backend: BackendNative, OCR: OCRNever, Tables: true})
			for _, want := range tests[name] {
				if !strings.Contains(got, want) {
					t.Errorf("output missing %q:\n%s", want, got)
				}
			}
		})
	}
}

func TestConvertOptions(t *testing.T) {
	opts := Options{Backend: BackendNative, OCR: OCRNever, Tables: true, Pages: "2", PageBreaks: true, FrontMatter: true}
	got := convertFixture(t, "basic", opts)
	if !strings.Contains(got, `title: "Risks"`) || !strings.Contains(got, "pages: 2") {
		t.Errorf("front matter:\n%s", got)
	}
	if strings.Contains(got, "Quarterly") {
		t.Errorf("page 1 content in output:\n%s", got)
	}

	if _, err := New(Options{Pages: "x"}); err == nil {
		t.Error("invalid pages: want error")
	}
	if _, err := New(Options{Backend: BackendMutool, MutoolPath: "/nonexistent/mutool"}); err == nil {
		t.Error("missing mutool: want error")
	}
}
