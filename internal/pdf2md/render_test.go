package pdf2md

import (
	"errors"
	"strings"
	"testing"
)

func TestFormatRuns(t *testing.T) {
	tests := []struct {
		name string
		runs []run
		want string
	}{
		{"plain escaped", []run{{text: "a*b_c [x] <y>"}}, `a\*b\_c \[x\] \<y>`},
		{"bold with spaces outside", []run{{text: "is"}, {text: " bold ", bold: true}, {text: "here"}}, "is **bold** here"},
		{"nested", []run{{text: "kilo", italic: true}, {text: "bits", italic: true, bold: true}, {text: " per second", italic: true}}, "*kilo**bits** per second*"},
		{"bold italic", []run{{text: "both", bold: true, italic: true}}, "***both***"},
		{"link", []run{{text: "see "}, {text: "the site", link: "https://x.org/a b"}, {text: "."}}, "see [the site](https://x.org/a%20b)."},
		{"styled link", []run{{text: "go", link: "u", bold: true}}, "[**go**](u)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatRuns(tt.runs); got != tt.want {
				t.Errorf("formatRuns = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInlineRuns(t *testing.T) {
	lines := []Line{
		newLine([]Span{sp("The plan is", 72, 100, 11), bold(sp("ambi-", 140, 100, 11))}),
		newLine([]Span{bold(sp("tious", 72, 114, 11)), sp("but", 110, 114, 11), italic(sp("fine", 135, 114, 11))}),
	}
	if got := formatRuns(inlineRuns(lines)); got != "The plan is **ambitious** but *fine*" {
		t.Errorf("inline = %q", got)
	}
}

func TestEscapeStart(t *testing.T) {
	tests := map[string]string{
		"# not heading": `\# not heading`,
		"- not list":    `\- not list`,
		"1. not list":   `1\. not list`,
		"12) not list":  `12\) not list`,
		"plain":         "plain",
	}
	for in, want := range tests {
		if got := escapeStart(in); got != want {
			t.Errorf("escapeStart(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRenderBlocks(t *testing.T) {
	tests := []struct {
		name string
		b    Block
		want string
	}{
		{"heading", Block{Kind: KindHeading, Level: 2, Lines: textLines(72, 0, 16, "A_b")}, "## A\\_b"},
		{"caption", Block{Kind: KindCaption, Lines: textLines(72, 0, 9, "Figure 1: x")}, "*Figure 1: x*"},
		{"image", Block{Kind: KindImage, Image: &ImageRef{Path: "a_images/p1_1.png", Alt: "Figure [1]"}}, "![Figure 1](a_images/p1_1.png)"},
		{"table", Block{Kind: KindTable, Rows: [][]string{{"a", "b"}, {"x|y"}}}, "| a | b |\n| --- | --- |\n| x\\|y |  |"},
		{
			"code",
			Block{Kind: KindCode, Lines: []Line{
				newLine([]Span{mono(sp("if x {", 72, 0, 10))}),
				newLine([]Span{mono(sp("y()", 96, 12, 10))}),
				newLine([]Span{mono(sp("}", 72, 24, 10))}),
			}},
			"```\nif x {\n    y()\n}\n```",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := renderBlock(tt.b); got != tt.want {
				t.Errorf("renderBlock = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderDoc(t *testing.T) {
	list := func(text string, level int) Block {
		return Block{Kind: KindList, Level: level, Lines: textLines(72, 0, 11, text)}
	}
	pages := []Page{
		{Num: 1, Blocks: []Block{
			list("• one", 0), list("◦ nested", 1), list("3. three", 2), list("b) letter", 0),
			para(textLines(72, 0, 11, "after")...),
		}},
		{Num: 2, Failed: errors.New("boom")},
		{Num: 3, Failed: errNoText},
	}
	var b strings.Builder
	fm := &FrontMatter{Title: `Say "hi"`, Pages: 3, Source: "x.pdf"}
	if err := renderDoc(&b, pages, fm, true); err != nil {
		t.Fatal(err)
	}
	want := `---
title: "Say \"hi\""
author: ""
pages: 3
source: "x.pdf"
---

- one
    - nested
        3. three
- b) letter

after

<!-- page 2 -->

<!-- page 2: text extraction failed -->

<!-- page 3 -->

<!-- page 3: no text -->
`
	if b.String() != want {
		t.Errorf("renderDoc =\n%s\nwant\n%s", b.String(), want)
	}
}
