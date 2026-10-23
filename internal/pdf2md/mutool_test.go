package pdf2md

import (
	"os"
	"testing"
)

func TestParseStext(t *testing.T) {
	f, err := os.Open("testdata/stext/styles.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	pages, err := parseStext(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 2 {
		t.Fatalf("pages = %d, want 2", len(pages))
	}
	p := pages[0]
	if p.Num != 3 || p.W != 612 || p.H != 792 {
		t.Errorf("page = %d %vx%v, want 3 612x792", p.Num, p.W, p.H)
	}
	if len(p.Images) != 1 || p.Images[0].BBox != (Rect{72, 400, 272, 600}) {
		t.Errorf("images = %+v", p.Images)
	}

	want := []struct {
		text               string
		font               string
		bold, italic, mono bool
	}{
		{"Hi ", "Helvetica", false, false, false},
		{"Bo", "Helvetica", true, false, false},
		{"f", "Times-Italic", false, true, false},
		{"x", "CourierNewPSMT", false, false, true},
	}
	if len(p.Spans) != len(want) {
		t.Fatalf("spans = %+v, want %d (rotated line dropped)", p.Spans, len(want))
	}
	for i, w := range want {
		t.Run(w.text, func(t *testing.T) {
			s := p.Spans[i]
			if s.Text != w.text || s.Font != w.font || s.Bold != w.bold || s.Italic != w.italic || s.Mono != w.mono {
				t.Errorf("span = %+v, want %+v", s, w)
			}
		})
	}
	s := p.Spans[0]
	if s.X != 72 || s.Y != 90 || s.W != 12 || s.H != 8 || s.Size != 10 {
		t.Errorf("geometry = x%v y%v w%v h%v size%v, want x72 y90 w12 h8 size10", s.X, s.Y, s.W, s.H, s.Size)
	}
	if len(pages[1].Spans) != 0 || pages[1].Num != 4 {
		t.Errorf("empty page = %+v", pages[1])
	}
}

func TestParseStextSplitsWideGaps(t *testing.T) {
	xml := `<document><page id="page1" width="100" height="100"><block><line dir="1 0"><font name="Helvetica" size="10">
<char c="a" quad="0 0 5 0 0 10 5 10" x="0" y="10"/>
<char c="b" quad="30 0 35 0 30 10 35 10" x="30" y="10"/>
</font></line></block></page></document>`
	pages, err := parseStext(stringsReader(xml))
	if err != nil {
		t.Fatal(err)
	}
	if got := len(pages[0].Spans); got != 2 {
		t.Errorf("spans = %d, want 2", got)
	}
}
