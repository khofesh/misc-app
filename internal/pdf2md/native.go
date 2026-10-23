package pdf2md

import (
	"context"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/ledongthuc/pdf"
)

// nativeExtractor is the pure Go fallback backend.
type nativeExtractor struct {
	password string
}

func (n nativeExtractor) open(path string) (r *pdf.Reader, f *os.File, err error) {
	f, err = os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, nil, err
	}
	tried := false
	r, err = pdf.NewReaderEncrypted(f, st.Size(), func() string {
		if tried {
			return ""
		}
		tried = true
		return n.password
	})
	if err != nil {
		f.Close()
		return nil, nil, fmt.Errorf("open %s: %w", path, err)
	}
	return r, f, nil
}

func (n nativeExtractor) PageCount(_ context.Context, path string) (int, error) {
	r, f, err := n.open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return r.NumPage(), nil
}

func (n nativeExtractor) Extract(ctx context.Context, path string, pages []int) ([]Page, error) {
	r, f, err := n.open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	out := make([]Page, 0, len(pages))
	for _, num := range pages {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		out = append(out, nativePage(r, num))
	}
	return out, nil
}

func nativePage(r *pdf.Reader, num int) (pg Page) {
	pg.Num = num
	defer func() {
		if x := recover(); x != nil {
			pg.Spans = nil
			pg.Failed = fmt.Errorf("native backend: %v", x)
		}
	}()

	p := r.Page(num)
	if p.V.IsNull() {
		pg.Failed = fmt.Errorf("native backend: page %d not found", num)
		return pg
	}
	x0, y1 := 0.0, 792.0
	pg.W, pg.H = 612, 792
	if mb := inherited(p.V, "MediaBox"); mb.Len() == 4 {
		x0, y1 = mb.Index(0).Float64(), mb.Index(3).Float64()
		pg.W = mb.Index(2).Float64() - x0
		pg.H = y1 - mb.Index(1).Float64()
	}

	var cur *Span
	flush := func() {
		if cur != nil && strings.TrimSpace(cur.Text) != "" {
			pg.Spans = append(pg.Spans, *cur)
		}
		cur = nil
	}
	for _, t := range p.Content().Text {
		if t.S == "" {
			continue
		}
		x, y := t.X-x0, y1-t.Y
		font := cleanFontName(t.Font)
		size := max(t.FontSize, 1)
		w := t.W
		sameRun := cur != nil && cur.Font == font && cur.Size == t.FontSize && abs(y-cur.Y) < 0.2*size
		if w <= 0 {
			// fonts without a Widths array (e.g. the standard 14) report no
			// width and no advance, so estimate both
			w = float64(utf8.RuneCountInString(t.S)) * 0.5 * size
			if sameRun && x < cur.X+cur.W {
				x = cur.X + cur.W
			}
		}
		if sameRun {
			gap := x - (cur.X + cur.W)
			if gap > -0.5*size && gap < 1.5*size {
				if gap > 0.25*size && !strings.HasSuffix(cur.Text, " ") && !strings.HasPrefix(t.S, " ") {
					cur.Text += " "
				}
				cur.Text += t.S
				cur.W = max(cur.W, x+w-cur.X)
				continue
			}
		}
		flush()
		bold, italic, mono := fontStyle(font)
		cur = &Span{Text: t.S, X: x, Y: y, W: w, H: t.FontSize * 0.75, Font: font, Size: t.FontSize, Bold: bold, Italic: italic, Mono: mono}
	}
	flush()
	return pg
}

func inherited(v pdf.Value, key string) pdf.Value {
	for ; !v.IsNull(); v = v.Key("Parent") {
		if r := v.Key(key); !r.IsNull() {
			return r
		}
	}
	return pdf.Value{}
}
