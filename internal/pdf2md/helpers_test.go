package pdf2md

import (
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestMain(m *testing.M) {
	log.Logger = zerolog.Nop()
	os.Exit(m.Run())
}

func stringsReader(s string) *strings.Reader { return strings.NewReader(s) }

// sp builds a span with an estimated width of half the font size per rune.
func sp(text string, x, y, size float64) Span {
	return Span{
		Text: text, X: x, Y: y, Size: size, Font: "Helvetica",
		W: float64(utf8.RuneCountInString(text)) * 0.5 * size, H: 0.7 * size,
	}
}

func bold(s Span) Span   { s.Bold = true; return s }
func italic(s Span) Span { s.Italic = true; return s }
func mono(s Span) Span   { s.Mono = true; return s }

// textLines builds one line per string at x, starting at y, 14pt apart.
func textLines(x, y, size float64, texts ...string) []Line {
	out := make([]Line, len(texts))
	for i, t := range texts {
		out[i] = newLine([]Span{sp(t, x, y+float64(i)*14, size)})
	}
	return out
}

func blockTexts(bs []Block) []string {
	out := make([]string, len(bs))
	for i, b := range bs {
		out[i] = b.Text()
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
