package pdf2md

import (
	"math"
	"regexp"
	"slices"
	"sort"
)

// frag is a piece of a row with no gap wider than fragGap × font size.
// Fragments are the unit for header/footer removal and column detection.
type frag struct {
	Line
	row  int     // -1 for images
	rowY float64 // vertical position of the row, for ordering
}

const fragGap = 1.5

func centerY(r Rect) float64 { return (r.Y0 + r.Y1) / 2 }

// buildFragments groups spans into rows by vertical center, then splits
// each row at wide horizontal gaps.
func buildFragments(spans []Span) []frag {
	if len(spans) == 0 {
		return nil
	}
	ss := slices.Clone(spans)
	sort.SliceStable(ss, func(i, j int) bool { return centerY(ss[i].BBox()) < centerY(ss[j].BBox()) })

	var rows [][]Span
	var first Span
	for _, s := range ss {
		if len(rows) > 0 && abs(centerY(s.BBox())-centerY(first.BBox())) < 0.5*max(s.H, first.H) {
			rows[len(rows)-1] = append(rows[len(rows)-1], s)
			continue
		}
		rows = append(rows, []Span{s})
		first = s
	}

	var out []frag
	for ri, row := range rows {
		sort.SliceStable(row, func(i, j int) bool { return row[i].X < row[j].X })
		start := 0
		for i := 1; i <= len(row); i++ {
			if i < len(row) {
				prev := row[i-1]
				gap := row[i].X - (prev.X + prev.W)
				if gap < fragGap*max(prev.Size, row[i].Size) {
					continue
				}
			}
			out = append(out, frag{Line: newLine(row[start:i]), row: ri, rowY: centerY(row[0].BBox())})
			start = i
		}
	}
	return out
}

// imageFrags turns extracted images into placeholder fragments so they take
// part in reading order.
func imageFrags(imgs []ImageRef) []frag {
	var out []frag
	for i := range imgs {
		if imgs[i].Path == "" {
			continue
		}
		out = append(out, frag{Line: Line{BBox: imgs[i].BBox, Image: &imgs[i]}, row: -1, rowY: centerY(imgs[i].BBox)})
	}
	return out
}

// detectColumns returns the gutters between columns, or nil for a single column.
func detectColumns(frags []frag, body float64) [][2]float64 {
	if len(frags) < 6 || body <= 0 {
		return nil
	}
	minX, maxX := math.Inf(1), math.Inf(-1)
	minY, maxY := math.Inf(1), math.Inf(-1)
	for _, f := range frags {
		minX, maxX = min(minX, f.BBox.X0), max(maxX, f.BBox.X1)
		minY, maxY = min(minY, f.BBox.Y0), max(maxY, f.BBox.Y1)
	}
	hb := maxY - minY
	if hb <= 0 || maxX-minX < 4*body {
		return nil
	}

	x0 := int(math.Floor(minX))
	cover := make([]float64, int(math.Ceil(maxX))-x0+1)
	for _, f := range frags {
		for x := int(math.Ceil(f.BBox.X0)); x < int(math.Floor(f.BBox.X1)); x++ {
			cover[x-x0] += f.BBox.H()
		}
	}

	var gutters [][2]float64
	for i := 0; i < len(cover); {
		if cover[i] > 0.4*hb {
			i++
			continue
		}
		j := i
		for j < len(cover) && cover[j] <= 0.4*hb {
			j++
		}
		a, b := float64(i+x0), float64(j+x0)
		if i > 0 && j < len(cover) && b-a >= 2*body {
			gutters = append(gutters, [2]float64{a, b})
		}
		i = j
	}
	if len(gutters) == 0 || len(gutters) > 2 {
		return nil
	}

	for c := range len(gutters) + 1 {
		n := 0
		top, bot := math.Inf(1), math.Inf(-1)
		left, right := math.Inf(1), math.Inf(-1)
		for _, f := range frags {
			if columnOf(f.BBox, gutters) == c {
				n++
				top, bot = min(top, f.BBox.Y0), max(bot, f.BBox.Y1)
				left, right = min(left, f.BBox.X0), max(right, f.BBox.X1)
			}
		}
		// narrow columns are page numbers in a table of contents or margin notes
		if n < 3 || bot-top < 0.6*hb || right-left < 0.2*(maxX-minX) {
			return nil
		}
	}
	return gutters
}

// columnOf returns the column a box belongs to, or -1 when it crosses a
// gutter into the next column. Ragged line ends inside a gutter are allowed.
func columnOf(r Rect, gutters [][2]float64) int {
	for i, g := range gutters {
		if r.X1 <= g[1]+1 {
			if i == 0 || r.X0 >= gutters[i-1][0]-1 {
				return i
			}
			return -1
		}
	}
	if r.X0 >= gutters[len(gutters)-1][0]-1 {
		return len(gutters)
	}
	return -1
}

// orderFlows splits fragments into flows (column runs and full-width runs)
// in reading order and merges fragments of the same row into lines.
func orderFlows(frags []frag, body float64) [][]Line {
	gutters := detectColumns(frags, body)
	if gutters == nil {
		return [][]Line{mergeRows(frags)}
	}
	col := func(f frag) int { return columnOf(f.BBox, gutters) }
	var full []frag
	for _, f := range frags {
		if col(f) < 0 {
			full = append(full, f)
		}
	}
	sort.SliceStable(full, func(i, j int) bool { return full[i].BBox.Y0 < full[j].BBox.Y0 })

	// band b holds column fragments between full-width fragments b-1 and b
	bands := make([][][]frag, len(full)+1)
	for b := range bands {
		bands[b] = make([][]frag, len(gutters)+1)
	}
	for _, f := range frags {
		c := col(f)
		if c < 0 {
			continue
		}
		cy := centerY(f.BBox)
		b := sort.Search(len(full), func(i int) bool { return centerY(full[i].BBox) > cy })
		bands[b][c] = append(bands[b][c], f)
	}

	var flows [][]Line
	var pending []frag
	for b := range bands {
		for _, cf := range bands[b] {
			if len(cf) == 0 {
				continue
			}
			if len(pending) > 0 {
				flows = append(flows, mergeRows(pending))
				pending = nil
			}
			flows = append(flows, mergeRows(cf))
		}
		if b < len(full) {
			pending = append(pending, full[b])
		}
	}
	if len(pending) > 0 {
		flows = append(flows, mergeRows(pending))
	}
	return flows
}

// mergeRows sorts fragments top to bottom and joins fragments that share a
// row into one line.
func mergeRows(frags []frag) []Line {
	fs := slices.Clone(frags)
	sort.SliceStable(fs, func(i, j int) bool {
		a, b := fs[i], fs[j]
		if a.rowY != b.rowY {
			return a.rowY < b.rowY
		}
		return a.BBox.X0 < b.BBox.X0
	})
	var out []Line
	for i, f := range fs {
		if i > 0 && f.row >= 0 && f.row == fs[i-1].row && out[len(out)-1].Image == nil {
			last := &out[len(out)-1]
			last.Spans = append(last.Spans, f.Spans...)
			last.BBox = last.BBox.Union(f.BBox)
			continue
		}
		out = append(out, f.Line)
	}
	return out
}

// leadingRatio is the median baseline distance divided by font size for
// consecutive body lines, used to tell paragraph gaps from line gaps.
func leadingRatio(pages []Page) float64 {
	var rs []float64
	for _, pg := range pages {
		for _, flow := range pg.Flows {
			for i := 1; i < len(flow); i++ {
				a, b := flow[i-1], flow[i]
				if a.Image != nil || b.Image != nil {
					continue
				}
				sa, sb := a.Size(), b.Size()
				if sa <= 0 || abs(sa-sb) > 0.1*sa {
					continue
				}
				r := (b.BBox.Y1 - a.BBox.Y1) / sa
				if r > 0.8 && r < 2.5 {
					rs = append(rs, r)
				}
			}
		}
	}
	if len(rs) == 0 {
		return 1.2
	}
	slices.Sort(rs)
	return rs[len(rs)/2]
}

var listMarkerRe = regexp.MustCompile(`^(?:[•◦▪–\-*·‣○■□●]\s|[•◦▪‣●■]|\(?\d{1,3}[.)]\s|[a-z][.)]\s)`)

func startsWithMarker(l Line) bool {
	return listMarkerRe.MatchString(l.Text())
}

type cell struct {
	X0, X1 float64
	Text   string
}

// cells splits a line at gaps of at least fragGap × font size.
func cells(l Line) []cell {
	var out []cell
	start := 0
	for i := 1; i <= len(l.Spans); i++ {
		if i < len(l.Spans) {
			a, b := l.Spans[i-1], l.Spans[i]
			if b.X-(a.X+a.W) < fragGap*max(a.Size, b.Size) {
				continue
			}
		}
		part := newLine(l.Spans[start:i])
		out = append(out, cell{part.BBox.X0, part.BBox.X1, part.Text()})
		start = i
	}
	return out
}

func aligned(a, b []cell, tol float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		ca, cb := (a[i].X0+a[i].X1)/2, (b[i].X0+b[i].X1)/2
		if abs(a[i].X0-b[i].X0) > tol && abs(a[i].X1-b[i].X1) > tol && abs(ca-cb) > tol {
			return false
		}
	}
	return true
}

// tableRuns finds runs of at least 3 lines with the same number of aligned
// cells. It returns, per line index, the end (exclusive) of the run starting
// there, or 0.
func tableRuns(lines []Line, lead float64) []int {
	runs := make([]int, len(lines))
	cs := make([][]cell, len(lines))
	for i, l := range lines {
		if l.Image == nil && !l.allMono() && !leaderRe.MatchString(l.Text()) {
			cs[i] = cells(l)
		}
	}
	for i := 0; i < len(lines); {
		if len(cs[i]) < 2 {
			i++
			continue
		}
		j := i + 1
		for j < len(lines) && len(cs[j]) >= 2 {
			size := max(lines[j].Size(), 1)
			if !aligned(cs[i], cs[j], 0.5*size) && !aligned(cs[j-1], cs[j], 0.5*size) {
				break
			}
			if lines[j].BBox.Y1-lines[j-1].BBox.Y1 > 3*lead*size {
				break
			}
			j++
		}
		if j-i >= 3 {
			runs[i] = j
			i = j
		} else {
			i++
		}
	}
	return runs
}

var leaderRe = regexp.MustCompile(`(\.\s?){4,}\s*\d+$`)

var sentenceEndRe = regexp.MustCompile(`[.!?:]["'”’)]?$`)

// buildBlocks splits a flow into blocks (§5.3) and detects tables (§5.5).
func buildBlocks(flow []Line, lead float64, tables bool) []Block {
	var runs []int
	if tables {
		runs = tableRuns(flow, lead)
	}
	left, right := math.Inf(1), math.Inf(-1)
	for _, l := range flow {
		left, right = min(left, l.BBox.X0), max(right, l.BBox.X1)
	}
	width := right - left

	var out []Block
	var cur *Block
	add := func(b Block) {
		out = append(out, b)
		cur = &out[len(out)-1]
	}

	for i := 0; i < len(flow); i++ {
		l := flow[i]
		if runs != nil && runs[i] > 0 {
			b := Block{Kind: KindTable, Lines: flow[i:runs[i]], BBox: l.BBox}
			for _, tl := range b.Lines {
				b.BBox = b.BBox.Union(tl.BBox)
				var row []string
				for _, c := range cells(tl) {
					row = append(row, c.Text)
				}
				b.Rows = append(b.Rows, row)
			}
			add(b)
			cur = nil
			i = runs[i] - 1
			continue
		}
		if l.Image != nil {
			add(Block{Kind: KindImage, Image: l.Image, BBox: l.BBox})
			cur = nil
			continue
		}
		if cur != nil && !splitBefore(*cur, l, lead, right, width) {
			cur.Lines = append(cur.Lines, l)
			cur.BBox = cur.BBox.Union(l.BBox)
			continue
		}
		add(Block{Kind: KindParagraph, Lines: []Line{l}, BBox: l.BBox})
	}
	return out
}

func splitBefore(b Block, l Line, lead, right, width float64) bool {
	p := b.Lines[len(b.Lines)-1]
	first := b.Lines[0]
	ps, ls := p.Size(), l.Size()
	size := max(ps, ls, 1)

	d := l.BBox.Y1 - p.BBox.Y1
	if d <= 0 || d > 1.3*lead*size {
		return true
	}
	if abs(ps-ls) > 0.1*min(ps, ls) {
		return true
	}
	if p.allMono() != l.allMono() {
		return true
	}
	if p.allMono() {
		return false
	}
	if startsWithMarker(l) || leaderRe.MatchString(p.Text()) {
		return true
	}
	short := p.BBox.X1 < right-0.15*width
	if short && sentenceEndRe.MatchString(p.Text()) {
		return true
	}
	if short && p.allBold() && !l.allBold() {
		return true
	}

	dx := l.BBox.X0 - p.BBox.X0
	if startsWithMarker(first) {
		// hanging indent continuation of a list item
		return l.BBox.X0 < first.BBox.X0-0.5*size || dx > size && l.BBox.X0 > first.BBox.X0+4*size
	}
	if abs(dx) > size {
		return !(len(b.Lines) == 1 && dx < 0)
	}
	return false
}
