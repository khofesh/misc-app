package pdf2md

import (
	"math"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// bodySize is the font size covering the most characters in the document.
func bodySize(pages []Page) float64 {
	count := map[float64]int{}
	for _, pg := range pages {
		for _, f := range pg.frags {
			for _, s := range f.Spans {
				count[math.Round(s.Size*2)/2] += utf8.RuneCountInString(s.Text)
			}
		}
	}
	best, bestN := 0.0, 0
	for size, n := range count {
		if n > bestN || n == bestN && size < best {
			best, bestN = size, n
		}
	}
	return best
}

var (
	captionRe    = regexp.MustCompile(`^(Figure|Fig\.|Table)\s*\d+`)
	outlineNumRe = regexp.MustCompile(`^[\dIVXivx]+(\.[\d]+)*\.?\s+`)
)

func normTitle(s string) string {
	s = strings.ToLower(collapseSpace(s))
	return strings.TrimRight(s, ".:")
}

func hasLetter(s string) bool {
	return strings.IndexFunc(s, unicode.IsLetter) >= 0
}

type classifier struct {
	body      float64
	levels    []float64 // distinct heading sizes, largest first
	boldLevel int
	outline   map[string]int
}

func isSizeHeading(b Block, body float64) bool {
	if b.Kind != KindParagraph || len(b.Lines) > 4 {
		return false
	}
	t := b.Text()
	return b.Lines[0].Size() >= 1.15*body && len(strings.Fields(t)) <= 25 && hasLetter(t)
}

func isBoldHeading(b Block, body float64) bool {
	if b.Kind != KindParagraph || len(b.Lines) != 1 || !b.Lines[0].allBold() {
		return false
	}
	t := b.Text()
	return b.Lines[0].Size() < 1.15*body && len(strings.Fields(t)) <= 12 &&
		!strings.HasSuffix(t, ".") && hasLetter(t)
}

func newClassifier(pages []Page, body float64, outline map[string]int) *classifier {
	c := &classifier{body: body, outline: outline}
	var sizes []float64
	for _, pg := range pages {
		for _, b := range pg.Blocks {
			if isSizeHeading(b, body) {
				sizes = append(sizes, math.Round(b.Lines[0].Size()*2)/2)
			}
		}
	}
	slices.Sort(sizes)
	slices.Reverse(sizes)
	for _, s := range sizes {
		if n := len(c.levels); n == 0 || c.levels[n-1]-s > 0.05*s {
			c.levels = append(c.levels, s)
		}
	}
	c.boldLevel = min(len(c.levels)+1, 6)
	return c
}

func (c *classifier) headingLevel(size float64) int {
	size = math.Round(size*2) / 2
	for i, s := range c.levels {
		if s-size <= 0.05*s {
			return min(i+1, 6)
		}
	}
	return min(len(c.levels), 6)
}

func (c *classifier) outlineLevel(t string) (int, bool) {
	if len(c.outline) == 0 {
		return 0, false
	}
	n := normTitle(t)
	if lvl, ok := c.outline[n]; ok {
		return lvl, true
	}
	lvl, ok := c.outline[outlineNumRe.ReplaceAllString(n, "")]
	return lvl, ok
}

// classify sets the kind and level of every paragraph block (§5.4).
func (c *classifier) classify(blocks []Block) {
	for i := range blocks {
		b := &blocks[i]
		if b.Kind != KindParagraph {
			continue
		}
		t := b.Text()
		allMono := true
		for _, l := range b.Lines {
			allMono = allMono && l.allMono()
		}
		switch {
		case len(b.Lines) <= 4 && hasLetter(t) && c.outlineMatch(b, t):
		case isSizeHeading(*b, c.body):
			b.Kind, b.Level = KindHeading, c.headingLevel(b.Lines[0].Size())
		case allMono:
			b.Kind = KindCode
		case startsWithMarker(b.Lines[0]):
			b.Kind = KindList
		case isBoldHeading(*b, c.body):
			b.Kind, b.Level = KindHeading, c.boldLevel
		case captionRe.MatchString(t) && b.Lines[0].Size() < 0.95*c.body:
			b.Kind = KindCaption
		}
	}
	listLevels(blocks)
}

func (c *classifier) outlineMatch(b *Block, t string) bool {
	lvl, ok := c.outlineLevel(t)
	if ok {
		b.Kind, b.Level = KindHeading, min(lvl, 6)
	}
	return ok
}

// listLevels assigns nesting levels from the left indent within each run of
// consecutive list items.
func listLevels(blocks []Block) {
	for i := 0; i < len(blocks); {
		if blocks[i].Kind != KindList {
			i++
			continue
		}
		j := i
		var xs []float64
		for j < len(blocks) && blocks[j].Kind == KindList {
			xs = append(xs, blocks[j].Lines[0].BBox.X0)
			j++
		}
		slices.Sort(xs)
		var levels []float64
		for _, x := range xs {
			if len(levels) == 0 || x-levels[len(levels)-1] > 0.5*blocks[i].Lines[0].Size() {
				levels = append(levels, x)
			}
		}
		for k := i; k < j; k++ {
			x := blocks[k].Lines[0].BBox.X0
			lvl := 0
			for l, lx := range levels {
				if x >= lx-0.5*blocks[k].Lines[0].Size() {
					lvl = l
				}
			}
			blocks[k].Level = lvl
		}
		i = j
	}
}

// attachCaptions uses a caption directly below an image as its alt text.
func attachCaptions(blocks []Block, body float64) {
	for i := 0; i+1 < len(blocks); i++ {
		img, next := blocks[i], blocks[i+1]
		if img.Kind != KindImage || img.Image == nil {
			continue
		}
		if next.Kind == KindCaption || captionRe.MatchString(next.Text()) {
			if next.BBox.Y0-img.BBox.Y1 < 2*body {
				img.Image.Alt = next.Text()
			}
		}
	}
}
