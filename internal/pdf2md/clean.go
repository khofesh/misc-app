package pdf2md

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

var textReplacer = strings.NewReplacer(
	"ﬁ", "fi", "ﬂ", "fl", "ﬀ", "ff", "ﬃ", "ffi", "ﬄ", "ffl", "ﬅ", "st", "ﬆ", "st",
	"\u00ad", "", "\u200b", "", "\u200c", "", "\u200d", "", "\ufeff", "",
	"\u00a0", " ",
)

func normalizeText(s string) string {
	return norm.NFC.String(textReplacer.Replace(s))
}

func normalizeSpans(spans []Span) []Span {
	out := spans[:0]
	for _, s := range spans {
		s.Text = normalizeText(s.Text)
		if strings.TrimSpace(s.Text) != "" {
			out = append(out, s)
		}
	}
	return out
}

var (
	digitsRe     = regexp.MustCompile(`\d+`)
	pageNumberRe = regexp.MustCompile(`(?i)^(page\s+)?[-–]?\s*\d+\s*[-–]?(\s*(of|/)\s*\d+)?$`)
)

const marginZone = 0.08

// removeHeadersFooters drops running headers and footers and page numbers
// from the top and bottom margin zones (§5.6).
func removeHeadersFooters(pages []Page) {
	inZone := func(pg *Page, f frag) bool {
		return pg.H > 0 && (f.BBox.Y1 <= pg.H*marginZone || f.BBox.Y0 >= pg.H*(1-marginZone))
	}
	key := func(f frag) string {
		return strings.ToLower(digitsRe.ReplaceAllString(f.Text(), "#"))
	}

	seen := map[string]int{}
	textPages := 0
	for i := range pages {
		if len(pages[i].frags) > 0 {
			textPages++
		}
		keys := map[string]bool{}
		for _, f := range pages[i].frags {
			if f.Image == nil && inZone(&pages[i], f) {
				keys[key(f)] = true
			}
		}
		for k := range keys {
			seen[k]++
		}
	}

	for i := range pages {
		pg := &pages[i]
		out := pg.frags[:0]
		for _, f := range pg.frags {
			if f.Image == nil && inZone(pg, f) {
				if pageNumberRe.MatchString(f.Text()) {
					continue
				}
				if textPages >= 2 && 2*seen[key(f)] >= textPages {
					continue
				}
			}
			out = append(out, f)
		}
		pg.frags = out
	}
}

var hyphenEndRe = regexp.MustCompile(`\p{L}-$`)

// dehyphenate reports whether a line ending in a with the next line starting
// with b is a word broken by a hyphen.
func dehyphenate(a, b string) bool {
	r, _ := utf8.DecodeRuneInString(b)
	return hyphenEndRe.MatchString(a) && unicode.IsLower(r)
}

// joinLines joins the lines of a paragraph, removing line-break hyphens.
func joinLines(parts []string) string {
	var b strings.Builder
	for i, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		cur := b.String()
		switch {
		case i == 0 || cur == "":
		case dehyphenate(cur, p):
			s := strings.TrimSuffix(cur, "-")
			b.Reset()
			b.WriteString(s)
		default:
			b.WriteByte(' ')
		}
		b.WriteString(p)
	}
	return b.String()
}

var paraEndRe = regexp.MustCompile(`[.!?:;]["'”’)\]]?$`)

// mergeAcrossPages joins a paragraph that continues on the next page.
func mergeAcrossPages(pages []Page) {
	prev := -1
	for i := range pages {
		if len(pages[i].Blocks) == 0 {
			continue
		}
		if prev >= 0 {
			a := &pages[prev].Blocks[len(pages[prev].Blocks)-1]
			b := pages[i].Blocks[0]
			if a.Kind == KindParagraph && b.Kind == KindParagraph && !paraEndRe.MatchString(a.Text()) {
				r, _ := utf8.DecodeRuneInString(b.Text())
				if unicode.IsLower(r) {
					a.Lines = append(a.Lines, b.Lines...)
					pages[i].Blocks = pages[i].Blocks[1:]
				}
			}
		}
		if len(pages[i].Blocks) > 0 {
			prev = i
		}
	}
}
