package pdf2md

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"slices"
	"strings"
)

type run struct {
	text         string
	bold, italic bool
	link         string
}

// inlineRuns turns block lines into styled runs, joining lines with spaces
// and removing line-break hyphens.
func inlineRuns(lines []Line) []run {
	var runs []run
	for li, l := range lines {
		for si, s := range l.Spans {
			t := s.Text
			switch {
			case si > 0 && needsSpace(l.Spans[si-1], s):
				t = " " + t
			case si == 0 && li > 0 && len(runs) > 0:
				last := &runs[len(runs)-1]
				trimmed := strings.TrimRight(last.text, " \t")
				if dehyphenate(trimmed, strings.TrimLeft(t, " \t")) {
					last.text = strings.TrimSuffix(trimmed, "-")
					t = strings.TrimLeft(t, " \t")
				} else {
					t = " " + t
				}
			}
			runs = append(runs, run{t, s.Bold, s.Italic, s.Link})
		}
	}

	// collapse whitespace across runs and merge runs with the same style
	var out []run
	for _, r := range runs {
		r.text = spaceRe.ReplaceAllString(r.text, " ")
		if n := len(out); n > 0 && strings.HasSuffix(out[n-1].text, " ") {
			r.text = strings.TrimPrefix(r.text, " ")
		} else if n == 0 {
			r.text = strings.TrimLeft(r.text, " ")
		}
		if r.text == "" {
			continue
		}
		if n := len(out); n > 0 && out[n-1].bold == r.bold && out[n-1].italic == r.italic && out[n-1].link == r.link {
			out[n-1].text += r.text
			continue
		}
		out = append(out, r)
	}
	if n := len(out); n > 0 {
		out[n-1].text = strings.TrimRight(out[n-1].text, " ")
	}
	return out
}

// trimRuns removes the first n bytes of text from runs.
func trimRuns(runs []run, n int) []run {
	for len(runs) > 0 && n > 0 {
		if len(runs[0].text) > n {
			runs[0].text = runs[0].text[n:]
			break
		}
		n -= len(runs[0].text)
		runs = runs[1:]
	}
	if len(runs) > 0 {
		runs[0].text = strings.TrimLeft(runs[0].text, " ")
	}
	return runs
}

func runsText(runs []run) string {
	var b strings.Builder
	for _, r := range runs {
		b.WriteString(r.text)
	}
	return b.String()
}

// formatRuns renders runs with emphasis markers, nesting bold inside italic
// where they overlap and keeping whitespace outside the markers. Links are
// rendered as separate units.
func formatRuns(runs []run) string {
	var b strings.Builder
	for i := 0; i < len(runs); {
		j := i + 1
		for j < len(runs) && runs[j].link == runs[i].link {
			j++
		}
		if runs[i].link == "" {
			b.WriteString(emphasize(runs[i:j]))
		} else {
			inner := emphasize(runs[i:j])
			core := strings.TrimSpace(inner)
			lead := inner[:strings.Index(inner, core)]
			b.WriteString(lead + "[" + core + "](" + escapeURL(runs[i].link) + ")" + inner[len(lead)+len(core):])
		}
		i = j
	}
	return b.String()
}

func emphasize(runs []run) string {
	var b strings.Builder
	var stack []string // open markers, outermost first
	pending := ""      // whitespace held until markers are placed
	want := func(m string, r run) bool {
		return m == "*" && r.italic || m == "**" && r.bold
	}
	for _, r := range runs {
		core := strings.TrimSpace(r.text)
		if core == "" {
			pending += r.text
			continue
		}
		lead := r.text[:strings.Index(r.text, core)]
		trail := r.text[len(lead)+len(core):]

		keep := 0
		for keep < len(stack) && want(stack[keep], r) {
			keep++
		}
		for k := len(stack) - 1; k >= keep; k-- {
			b.WriteString(stack[k])
		}
		stack = stack[:keep]
		b.WriteString(pending + lead)
		for _, m := range []string{"*", "**"} {
			if want(m, r) && !slices.Contains(stack, m) {
				stack = append(stack, m)
				b.WriteString(m)
			}
		}
		b.WriteString(escapeInline(core))
		pending = trail
	}
	for k := len(stack) - 1; k >= 0; k-- {
		b.WriteString(stack[k])
	}
	b.WriteString(pending)
	return b.String()
}

var inlineEscaper = strings.NewReplacer(
	`\`, `\\`, "*", `\*`, "_", `\_`, "`", "\\`", "[", `\[`, "]", `\]`, "<", `\<`,
)

func escapeInline(s string) string { return inlineEscaper.Replace(s) }

func escapeURL(u string) string {
	return strings.NewReplacer(" ", "%20", "(", "%28", ")", "%29").Replace(u)
}

var orderedStartRe = regexp.MustCompile(`^(\d+)([.)])`)

// escapeStart escapes characters that would turn a paragraph into another
// block type.
func escapeStart(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '#', '-', '+', '>', '=', '|':
		return `\` + s
	}
	if m := orderedStartRe.FindStringSubmatchIndex(s); m != nil {
		return s[:m[4]] + `\` + s[m[4]:]
	}
	return s
}

var (
	bulletRe  = regexp.MustCompile(`^(?:[•◦▪–\-*·‣○■□●])\s*`)
	numberRe  = regexp.MustCompile(`^\(?(\d{1,3})[.)]\s*`)
	letterRe  = regexp.MustCompile(`^[a-z][.)]\s`)
	codeFence = regexp.MustCompile("(?m)^```")
)

func renderList(b Block) (prefix, body string) {
	runs := inlineRuns(b.Lines)
	t := runsText(runs)
	switch {
	case numberRe.MatchString(t):
		m := numberRe.FindStringSubmatch(t)
		return m[1] + ". ", formatRuns(trimRuns(runs, len(m[0])))
	case letterRe.MatchString(t):
		return "- ", formatRuns(runs)
	default:
		m := bulletRe.FindString(t)
		return "- ", formatRuns(trimRuns(runs, len(m)))
	}
}

func renderCode(b Block) string {
	minX := math.Inf(1)
	for _, l := range b.Lines {
		minX = min(minX, l.BBox.X0)
	}
	var lines []string
	for _, l := range b.Lines {
		cw := 0.6 * max(l.Size(), 1)
		var sb strings.Builder
		sb.WriteString(strings.Repeat(" ", int(math.Round((l.BBox.X0-minX)/cw))))
		for i, s := range l.Spans {
			if i > 0 {
				prev := l.Spans[i-1]
				if gap := s.X - (prev.X + prev.W); gap > 0.3*cw {
					sb.WriteString(strings.Repeat(" ", max(1, int(math.Round(gap/cw)))))
				}
			}
			sb.WriteString(s.Text)
		}
		lines = append(lines, strings.TrimRight(sb.String(), " "))
	}
	body := strings.Join(lines, "\n")
	fence := "```"
	if codeFence.MatchString(body) {
		fence = "~~~"
	}
	return fence + "\n" + body + "\n" + fence
}

func renderTable(b Block) string {
	cols := 0
	for _, r := range b.Rows {
		cols = max(cols, len(r))
	}
	row := func(cells []string) string {
		out := make([]string, cols)
		for i := range out {
			if i < len(cells) {
				out[i] = strings.ReplaceAll(escapeInline(collapseSpace(cells[i])), "|", `\|`)
			}
		}
		return "| " + strings.Join(out, " | ") + " |"
	}
	lines := []string{row(b.Rows[0]), "|" + strings.Repeat(" --- |", cols)}
	for _, r := range b.Rows[1:] {
		lines = append(lines, row(r))
	}
	return strings.Join(lines, "\n")
}

func renderBlock(b Block) string {
	switch b.Kind {
	case KindHeading:
		return strings.Repeat("#", max(b.Level, 1)) + " " + escapeInline(collapseSpace(b.Text()))
	case KindCode:
		return renderCode(b)
	case KindTable:
		return renderTable(b)
	case KindCaption:
		return "*" + escapeInline(collapseSpace(b.Text())) + "*"
	case KindImage:
		alt := strings.NewReplacer("[", "", "]", "").Replace(b.Image.Alt)
		return "![" + alt + "](" + escapeURL(b.Image.Path) + ")"
	}
	return escapeStart(formatRuns(inlineRuns(b.Lines)))
}

// FrontMatter is the optional YAML header.
type FrontMatter struct {
	Title, Author, Source string
	Pages                 int
}

func yamlString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func renderDoc(w io.Writer, pages []Page, fm *FrontMatter, pageBreaks bool) error {
	var parts []string
	if fm != nil {
		parts = append(parts, fmt.Sprintf("---\ntitle: %s\nauthor: %s\npages: %d\nsource: %s\n---",
			yamlString(fm.Title), yamlString(fm.Author), fm.Pages, yamlString(fm.Source)))
	}

	listLevel := -1
	for i, pg := range pages {
		if pageBreaks && i > 0 {
			parts = append(parts, fmt.Sprintf("<!-- page %d -->", pg.Num))
			listLevel = -1
		}
		if pg.Failed != nil {
			msg := "text extraction failed"
			if errors.Is(pg.Failed, errNoText) {
				msg = "no text"
			}
			parts = append(parts, fmt.Sprintf("<!-- page %d: %s -->", pg.Num, msg))
			listLevel = -1
			continue
		}
		for _, b := range pg.Blocks {
			if b.Kind != KindList {
				listLevel = -1
				parts = append(parts, renderBlock(b))
				continue
			}
			lvl := min(b.Level, listLevel+1)
			prefix, body := renderList(b)
			item := strings.Repeat("    ", lvl) + prefix + body
			if listLevel >= 0 {
				parts[len(parts)-1] += "\n" + item
			} else {
				parts = append(parts, item)
			}
			listLevel = lvl
		}
	}

	out := strings.Join(parts, "\n\n")
	if out != "" {
		out += "\n"
	}
	_, err := io.WriteString(w, out)
	return err
}
