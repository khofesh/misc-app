package htmltag

import (
	"strings"
	"testing"
)

func TestHtmlToText(t *testing.T) {
	tests := []struct {
		name        string
		html        string
		contains    []string
		notContains []string
	}{
		{
			name:     "heading is uppercased and underlined",
			html:     `<h1>Title</h1>`,
			contains: []string{"TITLE\n====="},
		},
		{
			name:     "paragraph text is kept",
			html:     `<p>Hello world</p>`,
			contains: []string{"Hello world"},
		},
		{
			name:     "list items get bullets",
			html:     `<ul><li>One</li><li>Two</li></ul>`,
			contains: []string{"• One", "• Two"},
		},
		{
			name:     "link includes href",
			html:     `<a href="https://example.com">Example</a>`,
			contains: []string{"Example (https://example.com)"},
		},
		{
			name:        "link without href has no parentheses",
			html:        `<a>Example</a>`,
			contains:    []string{"Example"},
			notContains: []string{"("},
		},
		{
			name:        "script and style are removed",
			html:        `<script>var secret = 1</script><style>p { color: red }</style><p>Body</p>`,
			contains:    []string{"Body"},
			notContains: []string{"secret", "color"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := HtmlToText(tt.html)
			if err != nil {
				t.Fatalf("HtmlToText() error = %v", err)
			}
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("HtmlToText() = %q, want it to contain %q", got, want)
				}
			}
			for _, unwanted := range tt.notContains {
				if strings.Contains(got, unwanted) {
					t.Errorf("HtmlToText() = %q, want it not to contain %q", got, unwanted)
				}
			}
		})
	}
}

func TestHtmlToTextEmpty(t *testing.T) {
	got, err := HtmlToText("")
	if err != nil {
		t.Fatalf("HtmlToText() error = %v", err)
	}
	if got != "" {
		t.Errorf("HtmlToText() = %q, want empty string", got)
	}
}

func TestHtmlToTextNoBlankLineRuns(t *testing.T) {
	got, err := HtmlToText(`<h1>A</h1><p>B</p><p>C</p><ul><li>D</li></ul>`)
	if err != nil {
		t.Fatalf("HtmlToText() error = %v", err)
	}
	if strings.Contains(got, "\n\n\n") {
		t.Errorf("HtmlToText() = %q, want no runs of more than one blank line", got)
	}
	for line := range strings.SplitSeq(got, "\n") {
		if line != strings.TrimSpace(line) {
			t.Errorf("line %q has leading or trailing whitespace", line)
		}
	}
}

func TestHtmlToTextNoDuplicateText(t *testing.T) {
	// regression: every element used to write its full s.Text(), so nested
	// text was emitted once per ancestor.
	tests := []struct {
		html string
		want string
	}{
		{`<p>Hello</p>`, "Hello"},
		{`<html><body><div><p>Hello</p></div></body></html>`, "Hello"},
		{`<ul><li>One</li><li>Two</li></ul>`, "• One\n• Two"},
		{`<a href="https://x.com">X</a>`, "X (https://x.com)"},
		{`<div>A</div><span>B</span>`, "A\nB"},
		{`<h1>Title</h1>`, "TITLE\n====="},
	}

	for _, tt := range tests {
		t.Run(tt.html, func(t *testing.T) {
			got, err := HtmlToText(tt.html)
			if err != nil {
				t.Fatalf("HtmlToText() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("HtmlToText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHtmlToTextLayout(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "inline elements keep word spacing",
			html: `<p>Some <strong>bold</strong> and <em>italic</em>text.</p>`,
			want: "Some bold and italictext.",
		},
		{
			name: "whitespace inside text is collapsed",
			html: "<p>  lots   of\n\t space  </p>",
			want: "lots of space",
		},
		{
			name: "paragraphs separated by one blank line",
			html: `<p>A</p><p>B</p>`,
			want: "A\n\nB",
		},
		{
			name: "heading then paragraph",
			html: `<h2>Intro</h2><p>Text</p>`,
			want: "INTRO\n=====\n\nText",
		},
		{
			name: "heading underline counts characters not bytes",
			html: `<h1>Привет</h1>`,
			want: "ПРИВЕТ\n======",
		},
		{
			name: "br breaks the line",
			html: `<p>line one<br>line two</p>`,
			want: "line one\nline two",
		},
		{
			name: "link inside list item",
			html: `<ul><li>See <a href="/docs">docs</a> here</li></ul>`,
			want: "• See docs (/docs) here",
		},
		{
			name: "nested lists",
			html: `<ul><li>A<ul><li>A1</li></ul></li><li>B</li></ul>`,
			want: "• A\n• A1\n• B",
		},
		{
			name: "table cells separated by spaces, rows by lines",
			html: `<table><tr><th>Name</th><th>Age</th></tr><tr><td>Ann</td><td>30</td></tr></table>`,
			want: "Name Age\nAnn 30",
		},
		{
			name: "comments are skipped",
			html: `<p>visible<!-- hidden --></p>`,
			want: "visible",
		},
		{
			name: "nbsp-only block becomes a blank line",
			html: `<div>verse one</div><div>&nbsp;</div><div>verse two</div>`,
			want: "verse one\n\nverse two",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := HtmlToText(tt.html)
			if err != nil {
				t.Fatalf("HtmlToText() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("HtmlToText() = %q, want %q", got, tt.want)
			}
		})
	}
}
