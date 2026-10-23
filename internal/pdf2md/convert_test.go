package pdf2md

import (
	"fmt"
	"testing"
)

func TestParsePageRanges(t *testing.T) {
	tests := []struct {
		spec string
		n    int
		want string
		err  bool
	}{
		{"", 3, "[1 2 3]", false},
		{"1-3,7,10-", 12, "[1 2 3 7 10 11 12]", false},
		{"2, 2,1", 5, "[1 2]", false},
		{"4-9", 5, "[4 5]", false},
		{"0", 5, "", true},
		{"3-1", 5, "", true},
		{"a", 5, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			r, err := parsePageRanges(tt.spec)
			if (err != nil) != tt.err {
				t.Fatalf("err = %v, want error %v", err, tt.err)
			}
			if err == nil {
				if got := fmt.Sprint(selectPages(r, tt.n)); got != tt.want {
					t.Errorf("pages = %s, want %s", got, tt.want)
				}
			}
		})
	}
}

func TestFormatPages(t *testing.T) {
	if got := formatPages([]int{1, 2, 3, 5, 7, 8}); got != "1-3,5,7-8" {
		t.Errorf("formatPages = %q", got)
	}
}

func TestApplyLinks(t *testing.T) {
	s := sp("Read the MuPDF website now", 0, 100, 10)
	// "MuPDF website" covers runes 9-22 at 5pt per rune, give a loose rect
	l := link{R: Rect{43, 90, 113, 101}, URI: "https://mupdf.com"}
	got := applyLinks([]Span{s}, []link{l})
	var texts []string
	for _, g := range got {
		texts = append(texts, fmt.Sprintf("%s|%s", g.Text, g.Link))
	}
	want := []string{"Read the |", "MuPDF website|https://mupdf.com", " now|"}
	if !equalStrings(texts, want) {
		t.Errorf("spans = %q, want %q", texts, want)
	}
}

func TestParseHOCR(t *testing.T) {
	hocr := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html><body><div class='ocr_page' title='bbox 0 0 1000 1000'>
<span class='ocr_line' title="bbox 100 200 600 240; baseline 0 -10; x_size 40; x_descenders 10; x_ascenders 10">
<span class='ocrx_word' title='bbox 100 200 300 230; x_wconf 95'>Hello&amp;</span>
<span class='ocrx_word' title='bbox 320 200 600 232; x_wconf 90'><strong>World</strong></span>
</span></div></body></html>`
	spans, err := parseHOCR(stringsReader(hocr), 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if len(spans) != 2 {
		t.Fatalf("spans = %+v", spans)
	}
	s := spans[0]
	if s.Text != "Hello&" || s.X != 50 || s.Y != 115 || s.W != 100 || s.Size != 20 || s.H != 15 {
		t.Errorf("span 0 = %+v", s)
	}
	if !spans[1].Bold || spans[1].Text != "World" {
		t.Errorf("span 1 = %+v", spans[1])
	}
}

func TestNeedsOCR(t *testing.T) {
	scan := Page{W: 100, H: 100, Images: []ImageRef{{BBox: Rect{0, 0, 100, 80}}}}
	if !needsOCR(&scan) {
		t.Error("scanned page: want OCR")
	}
	scan.Spans = []Span{sp("plenty of text on this page already", 0, 10, 10)}
	if needsOCR(&scan) {
		t.Error("page with text: want no OCR")
	}
	if needsOCR(&Page{W: 100, H: 100, Images: []ImageRef{{BBox: Rect{0, 0, 10, 10}}}}) {
		t.Error("small image: want no OCR")
	}
}

func TestFontStyle(t *testing.T) {
	tests := []struct {
		name               string
		bold, italic, mono bool
	}{
		{"Helvetica-Bold", true, false, false},
		{"Times-BoldItalic", true, true, false},
		{"MinionPro-It", false, true, false},
		{"Arial-BoldIt", true, true, false},
		{"DejaVuSansMono", false, false, true},
		{"CMTT10", false, false, true},
		{"Helvetica", false, false, false},
	}
	for _, tt := range tests {
		b, i, m := fontStyle(tt.name)
		if b != tt.bold || i != tt.italic || m != tt.mono {
			t.Errorf("fontStyle(%q) = %v %v %v, want %v %v %v", tt.name, b, i, m, tt.bold, tt.italic, tt.mono)
		}
	}
	if got := cleanFontName("ABCDEF+Times-Roman"); got != "Times-Roman" {
		t.Errorf("cleanFontName = %q", got)
	}
}
