package pdf2md

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Rect is an axis-aligned box in PDF points with a top-left origin.
type Rect struct{ X0, Y0, X1, Y1 float64 }

func (r Rect) W() float64 { return r.X1 - r.X0 }
func (r Rect) H() float64 { return r.Y1 - r.Y0 }

func (r Rect) Union(o Rect) Rect {
	return Rect{min(r.X0, o.X0), min(r.Y0, o.Y0), max(r.X1, o.X1), max(r.Y1, o.Y1)}
}

func (r Rect) Overlaps(o Rect) bool {
	return r.X0 < o.X1 && o.X0 < r.X1 && r.Y0 < o.Y1 && o.Y0 < r.Y1
}

// Span is a run of text with one font, size and style.
type Span struct {
	Text   string
	X, Y   float64 // baseline origin, top-left page origin
	W, H   float64 // H is the height above the baseline
	Font   string
	Size   float64
	Bold   bool
	Italic bool
	Mono   bool
	Link   string
}

func (s Span) BBox() Rect { return Rect{s.X, s.Y - s.H, s.X + s.W, s.Y} }

// Line is a row of spans in one column, sorted by X. Image is set for
// image placeholders, which have no spans.
type Line struct {
	Spans []Span
	BBox  Rect
	Image *ImageRef
}

func newLine(spans []Span) Line {
	l := Line{Spans: spans, BBox: spans[0].BBox()}
	for _, s := range spans[1:] {
		l.BBox = l.BBox.Union(s.BBox())
	}
	return l
}

// Size is the font size covering the most characters in the line.
func (l Line) Size() float64 {
	count := map[float64]int{}
	best, bestN := 0.0, -1
	for _, s := range l.Spans {
		count[s.Size] += utf8.RuneCountInString(s.Text)
		if n := count[s.Size]; n > bestN || n == bestN && s.Size > best {
			best, bestN = s.Size, n
		}
	}
	return best
}

func (l Line) Text() string {
	var b strings.Builder
	for i, s := range l.Spans {
		if i > 0 && needsSpace(l.Spans[i-1], s) {
			b.WriteByte(' ')
		}
		b.WriteString(s.Text)
	}
	return collapseSpace(b.String())
}

func (l Line) allMono() bool {
	for _, s := range l.Spans {
		if !s.Mono && strings.TrimSpace(s.Text) != "" {
			return false
		}
	}
	return len(l.Spans) > 0
}

func (l Line) allBold() bool {
	for _, s := range l.Spans {
		if !s.Bold && strings.TrimSpace(s.Text) != "" {
			return false
		}
	}
	return len(l.Spans) > 0
}

// needsSpace reports whether a gap between two adjacent spans is a word break.
func needsSpace(a, b Span) bool {
	if strings.HasSuffix(a.Text, " ") || strings.HasPrefix(b.Text, " ") {
		return false
	}
	return b.X-(a.X+a.W) > 0.15*max(a.Size, b.Size)
}

var spaceRe = regexp.MustCompile(`\s+`)

func collapseSpace(s string) string { return strings.TrimSpace(spaceRe.ReplaceAllString(s, " ")) }

type BlockKind int

const (
	KindParagraph BlockKind = iota
	KindHeading
	KindList
	KindCode
	KindTable
	KindCaption
	KindImage
)

type Block struct {
	Kind  BlockKind
	Lines []Line
	Level int // heading level, or list nesting level
	BBox  Rect
	Rows  [][]string // KindTable
	Image *ImageRef  // KindImage
}

func (b Block) Text() string {
	parts := make([]string, len(b.Lines))
	for i, l := range b.Lines {
		parts[i] = l.Text()
	}
	return joinLines(parts)
}

// ImageRef is an image's position on the page and, once extracted, its link.
type ImageRef struct {
	BBox Rect
	Path string
	Alt  string
}

type Page struct {
	Num    int
	W, H   float64
	Spans  []Span
	Images []ImageRef
	Flows  [][]Line // lines in reading order, one slice per column run
	Blocks []Block
	frags  []frag
	OCR    bool
	Failed error
}

func (p *Page) charCount() int {
	n := 0
	for _, s := range p.Spans {
		n += utf8.RuneCountInString(strings.TrimSpace(s.Text))
	}
	return n
}

// Extractor turns PDF pages into positioned spans. Pages that fail are
// returned with Failed set rather than as an error.
type Extractor interface {
	PageCount(ctx context.Context, path string) (int, error)
	Extract(ctx context.Context, path string, pages []int) ([]Page, error)
}

type Backend string

const (
	BackendAuto   Backend = "auto"
	BackendMutool Backend = "mutool"
	BackendNative Backend = "native"
)

// ErrMissingTool is returned when a required external tool is not installed.
var ErrMissingTool = errors.New("missing external tool")

// FindMutool resolves the mutool binary from the flag value, the MUTOOL
// environment variable, then PATH. An explicit flag value is not replaced by
// the others. It returns "" when none is found.
func FindMutool(flag string) string {
	candidates := []string{os.Getenv("MUTOOL"), "mutool"}
	if flag != "" {
		candidates = []string{flag}
	}
	for _, p := range candidates {
		if p == "" {
			continue
		}
		if path, err := exec.LookPath(p); err == nil {
			return path
		}
	}
	return ""
}

func newExtractor(b Backend, mutool, password string) (Extractor, Backend, error) {
	switch b {
	case BackendMutool:
		if mutool == "" {
			return nil, "", fmt.Errorf("backend mutool: %w: mutool not found (install mupdf or set -mutool)", ErrMissingTool)
		}
		return mutoolExtractor{bin: mutool, password: password}, BackendMutool, nil
	case BackendNative:
		return nativeExtractor{password: password}, BackendNative, nil
	case BackendAuto, "":
		if mutool != "" {
			return mutoolExtractor{bin: mutool, password: password}, BackendMutool, nil
		}
		return nativeExtractor{password: password}, BackendNative, nil
	}
	return nil, "", fmt.Errorf("unknown backend %q", b)
}

var (
	subsetRe = regexp.MustCompile(`^[A-Z]{6}\+`)
	boldRe   = regexp.MustCompile(`(?i)bold|black|heavy|semibold|demi`)
	italicRe = regexp.MustCompile(`(?i)italic|oblique|[-,](bold)?it$`)
	monoRe   = regexp.MustCompile(`(?i)mono|courier|consolas|menlo|inconsolata|typewriter|cmtt|code`)
)

func cleanFontName(name string) string { return subsetRe.ReplaceAllString(name, "") }

func fontStyle(name string) (bold, italic, mono bool) {
	return boldRe.MatchString(name), italicRe.MatchString(name), monoRe.MatchString(name)
}

// formatPages renders page numbers in mutool's range syntax, e.g. "1-3,7".
func formatPages(pages []int) string {
	var parts []string
	for i := 0; i < len(pages); {
		j := i
		for j+1 < len(pages) && pages[j+1] == pages[j]+1 {
			j++
		}
		if i == j {
			parts = append(parts, fmt.Sprint(pages[i]))
		} else {
			parts = append(parts, fmt.Sprintf("%d-%d", pages[i], pages[j]))
		}
		i = j + 1
	}
	return strings.Join(parts, ",")
}
