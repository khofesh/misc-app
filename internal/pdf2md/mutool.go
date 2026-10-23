package pdf2md

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

const stextBold = 8 // FZ_STEXT_BOLD char flag

type mutoolExtractor struct {
	bin      string
	password string
}

func (m mutoolExtractor) run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, m.bin, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("mutool %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

func (m mutoolExtractor) withPassword(cmd string, rest ...string) []string {
	args := []string{cmd}
	if m.password != "" {
		args = append(args, "-p", m.password)
	}
	return append(args, rest...)
}

func (m mutoolExtractor) PageCount(ctx context.Context, path string) (int, error) {
	out, err := m.run(ctx, m.withPassword("pages", path)...)
	if err != nil {
		return 0, err
	}
	n := bytes.Count(out, []byte("<page pagenum="))
	if n == 0 {
		return 0, fmt.Errorf("mutool pages: no pages in %s", path)
	}
	return n, nil
}

func (m mutoolExtractor) Extract(ctx context.Context, path string, pages []int) ([]Page, error) {
	args := m.withPassword("draw", "-q", "-F", "stext", "-O", "preserve-images", "-o", "-", path, formatPages(pages))
	out, err := m.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	return parseStext(bytes.NewReader(out))
}

type stextParser struct {
	pages   []Page
	cur     *Page
	font    string
	size    float64
	skip    bool // inside a rotated line
	span    *Span
	spanTop float64
}

// parseStext reads mutool's structured text XML into pages of spans.
func parseStext(r io.Reader) ([]Page, error) {
	d := xml.NewDecoder(r)
	p := &stextParser{}
	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse stext: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			p.start(t)
		case xml.EndElement:
			switch t.Name.Local {
			case "line", "font":
				p.flush()
			case "page":
				p.flush()
				p.cur = nil
			}
		}
	}
	return p.pages, nil
}

func attr(e xml.StartElement, name string) string {
	for _, a := range e.Attr {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

func floats(s string) []float64 {
	var out []float64
	for _, f := range strings.Fields(s) {
		v, err := strconv.ParseFloat(f, 64)
		if err != nil {
			return nil
		}
		out = append(out, v)
	}
	return out
}

func bboxAttr(e xml.StartElement) (Rect, bool) {
	v := floats(attr(e, "bbox"))
	if len(v) != 4 {
		return Rect{}, false
	}
	return Rect{v[0], v[1], v[2], v[3]}, true
}

func (p *stextParser) start(e xml.StartElement) {
	switch e.Name.Local {
	case "page":
		num, _ := strconv.Atoi(strings.TrimPrefix(attr(e, "id"), "page"))
		w, _ := strconv.ParseFloat(attr(e, "width"), 64)
		h, _ := strconv.ParseFloat(attr(e, "height"), 64)
		p.pages = append(p.pages, Page{Num: num, W: w, H: h})
		p.cur = &p.pages[len(p.pages)-1]
	case "image":
		if r, ok := bboxAttr(e); ok && p.cur != nil {
			p.cur.Images = append(p.cur.Images, ImageRef{BBox: r})
		}
	case "block":
		if attr(e, "type") == "image" {
			if r, ok := bboxAttr(e); ok && p.cur != nil {
				p.cur.Images = append(p.cur.Images, ImageRef{BBox: r})
			}
		}
	case "line":
		dir := floats(attr(e, "dir"))
		p.skip = len(dir) == 2 && (dir[0] < 0.99 || dir[1] > 0.01 || dir[1] < -0.01)
	case "font":
		p.flush()
		p.font = cleanFontName(attr(e, "name"))
		p.size, _ = strconv.ParseFloat(attr(e, "size"), 64)
	case "char":
		if !p.skip && p.cur != nil {
			p.char(e)
		}
	}
}

func (p *stextParser) char(e xml.StartElement) {
	q := floats(attr(e, "quad"))
	if len(q) != 8 {
		return
	}
	x0, x1 := min(q[0], q[2], q[4], q[6]), max(q[0], q[2], q[4], q[6])
	top := min(q[1], q[3], q[5], q[7])
	y, err := strconv.ParseFloat(attr(e, "y"), 64)
	if err != nil {
		y = max(q[1], q[3], q[5], q[7])
	}
	flags, _ := strconv.Atoi(attr(e, "flags"))
	bold, italic, mono := fontStyle(p.font)
	bold = bold || flags&stextBold != 0
	c := attr(e, "c")
	size := max(p.size, 1)

	if s := p.span; s != nil {
		gap := x0 - (s.X + s.W)
		if s.Bold == bold && gap < 1.5*size && gap > -size && abs(y-s.Y) < 0.3*size {
			s.Text += c
			s.W = max(s.W, x1-s.X)
			p.spanTop = min(p.spanTop, top)
			return
		}
		p.flush()
	}
	p.span = &Span{Text: c, X: x0, Y: y, W: x1 - x0, Font: p.font, Size: p.size, Bold: bold, Italic: italic, Mono: mono}
	p.spanTop = top
}

func (p *stextParser) flush() {
	s := p.span
	p.span = nil
	if s == nil || p.cur == nil || strings.TrimSpace(s.Text) == "" {
		return
	}
	s.H = max(s.Y-p.spanTop, s.Size*0.5)
	p.cur.Spans = append(p.cur.Spans, *s)
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
