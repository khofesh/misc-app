package pdf2md

import (
	"context"
	"fmt"
	"os"
	"unicode"
	"unicode/utf8"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func pdfcpuConf(password string) *model.Configuration {
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed
	conf.UserPW = password
	return conf
}

type link struct {
	R   Rect
	URI string
}

// meta is the backend-independent document information from pdfcpu (§4.4).
type meta struct {
	Title, Author string
	Outline       map[string]int // normalized title → depth
	Links         map[int][]link // page → links in top-left page coordinates
}

func loadMeta(ctx context.Context, path, password string, pages []int) (meta, error) {
	m := meta{Outline: map[string]int{}, Links: map[int][]link{}}
	f, err := os.Open(path)
	if err != nil {
		return m, err
	}
	defer f.Close()

	conf := pdfcpuConf(password)
	conf.Cmd = model.LISTANNOTATIONS
	pc, err := api.ReadValidateAndOptimize(ctx, f, conf, nil)
	if err != nil {
		return m, fmt.Errorf("read metadata: %w", err)
	}
	m.Title, m.Author = pc.Title, pc.Author

	if bms, err := pdfcpu.Bookmarks(ctx, pc); err == nil {
		var walk func([]pdfcpu.Bookmark, int)
		walk = func(bms []pdfcpu.Bookmark, depth int) {
			for _, b := range bms {
				if t := normTitle(b.Title); t != "" {
					if _, ok := m.Outline[t]; !ok {
						m.Outline[t] = depth
					}
				}
				walk(b.Kids, depth+1)
			}
		}
		walk(bms, 1)
	}

	sel := types.IntSet{}
	for _, p := range pages {
		sel[p] = true
	}
	bounds, err := pc.PageBoundaries(ctx, sel)
	if err != nil {
		return m, nil
	}
	annots, err := pdfcpu.AnnotationsForSelectedPages(ctx, pc, sel)
	if err != nil {
		return m, nil
	}
	for pg, pa := range annots {
		if pg < 1 || pg > len(bounds) {
			continue
		}
		crop := bounds[pg-1].CropBox()
		if crop == nil {
			continue
		}
		for _, a := range pa[model.AnnLink].Map {
			var la model.LinkAnnotation
			switch v := a.(type) {
			case model.LinkAnnotation:
				la = v
			case *model.LinkAnnotation:
				la = *v
			default:
				continue
			}
			if la.URI == "" {
				continue
			}
			r := la.Rect
			m.Links[pg] = append(m.Links[pg], link{
				R:   Rect{r.LL.X - crop.LL.X, crop.UR.Y - r.UR.Y, r.UR.X - crop.LL.X, crop.UR.Y - r.LL.Y},
				URI: la.URI,
			})
		}
	}
	return m, nil
}

// applyLinks marks the parts of spans covered by link rectangles, splitting
// spans at the estimated character boundaries.
func applyLinks(spans []Span, links []link) []Span {
	for _, l := range links {
		var out []Span
		for _, s := range spans {
			b := s.BBox()
			ov := min(b.Y1, l.R.Y1) - max(b.Y0, l.R.Y0)
			n := utf8.RuneCountInString(s.Text)
			if s.Link != "" || n == 0 || s.W <= 0 || ov < 0.5*b.H() || !b.Overlaps(l.R) {
				out = append(out, s)
				continue
			}
			cw := s.W / float64(n)
			i0 := clamp(int((l.R.X0-s.X)/cw+0.5), 0, n)
			i1 := clamp(int((l.R.X1-s.X)/cw+0.5), 0, n)
			rs := []rune(s.Text)
			i0, i1 = snapWord(rs, i0, true), snapWord(rs, i1, false)
			if i1 <= i0 {
				out = append(out, s)
				continue
			}
			for _, part := range [][2]int{{0, i0}, {i0, i1}, {i1, n}} {
				if part[1] <= part[0] {
					continue
				}
				p := s
				p.Text = string(rs[part[0]:part[1]])
				p.X = s.X + float64(part[0])*cw
				p.W = float64(part[1]-part[0]) * cw
				if part[0] == i0 {
					p.Link = l.URI
				}
				out = append(out, p)
			}
		}
		spans = out
	}
	return spans
}

// snapWord moves a split index to the nearest word boundary, skipping the
// spaces next to it so they stay outside the link.
func snapWord(rs []rune, i int, start bool) int {
	boundary := func(j int) bool {
		return j <= 0 || j >= len(rs) || unicode.IsSpace(rs[j-1]) != unicode.IsSpace(rs[j])
	}
	for d := 0; d <= len(rs); d++ {
		for _, j := range []int{i - d, i + d} {
			if j < 0 || j > len(rs) || !boundary(j) {
				continue
			}
			if start && j < len(rs) && unicode.IsSpace(rs[j]) || !start && j > 0 && unicode.IsSpace(rs[j-1]) {
				continue
			}
			return j
		}
	}
	return i
}

func clamp(v, lo, hi int) int { return max(lo, min(v, hi)) }
