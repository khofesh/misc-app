package pdf2md

import "testing"

func para(lines ...Line) Block {
	b := Block{Kind: KindParagraph, Lines: lines, BBox: lines[0].BBox}
	for _, l := range lines[1:] {
		b.BBox = b.BBox.Union(l.BBox)
	}
	return b
}

func TestBodySize(t *testing.T) {
	pages := []Page{{frags: buildFragments([]Span{
		sp("Big title", 72, 50, 24),
		sp("lots of body text in eleven point", 72, 100, 11),
		sp("more body text", 72, 114, 11),
	})}}
	if got := bodySize(pages); got != 11 {
		t.Errorf("bodySize = %v, want 11", got)
	}
}

func TestClassify(t *testing.T) {
	body := "regular body text that is long enough to be a paragraph and not a heading."
	blocks := []Block{
		para(textLines(72, 50, 24, "Title")...),
		para(textLines(72, 90, 16, "Section")...),
		para(textLines(72, 120, 11, body)...),
		para(newLine([]Span{bold(sp("Bold heading", 72, 150, 11))})),
		para(textLines(72, 170, 11, "• first")...),
		para(textLines(72, 184, 11, "• second")...),
		para(textLines(90, 198, 11, "◦ nested")...),
		para(textLines(72, 212, 11, "2) numbered")...),
		para(newLine([]Span{mono(sp("x := 1", 72, 240, 10))})),
		para(textLines(72, 260, 9, "Figure 3: a chart")...),
		para(newLine([]Span{bold(sp("Ends with a period.", 72, 280, 11))})),
	}
	pages := []Page{{Blocks: blocks}}
	c := newClassifier(pages, 11, nil)
	c.classify(blocks)

	want := []struct {
		kind  BlockKind
		level int
	}{
		{KindHeading, 1}, {KindHeading, 2}, {KindParagraph, 0}, {KindHeading, 3},
		{KindList, 0}, {KindList, 0}, {KindList, 1}, {KindList, 0},
		{KindCode, 0}, {KindCaption, 0}, {KindParagraph, 0},
	}
	for i, w := range want {
		if blocks[i].Kind != w.kind || blocks[i].Level != w.level {
			t.Errorf("block %d %q = kind %v level %d, want kind %v level %d",
				i, blocks[i].Text(), blocks[i].Kind, blocks[i].Level, w.kind, w.level)
		}
	}
}

func TestClassifyOutline(t *testing.T) {
	blocks := []Block{
		para(textLines(72, 50, 24, "Title")...),
		para(textLines(72, 90, 16, "2.1 Methods")...),
		para(textLines(72, 120, 11, "Results")...),
	}
	c := newClassifier([]Page{{Blocks: blocks}}, 11, map[string]int{"methods": 1, "results": 3})
	c.classify(blocks)
	for i, want := range []int{1, 1, 3} {
		if blocks[i].Kind != KindHeading || blocks[i].Level != want {
			t.Errorf("block %d = kind %v level %d, want heading %d", i, blocks[i].Kind, blocks[i].Level, want)
		}
	}
}

func TestAttachCaptions(t *testing.T) {
	img := &ImageRef{BBox: Rect{72, 100, 272, 300}, Path: "x.png"}
	blocks := []Block{
		{Kind: KindImage, Image: img, BBox: img.BBox},
		{Kind: KindCaption, Lines: textLines(72, 312, 9, "Figure 1: Sales"), BBox: Rect{72, 305, 200, 312}},
	}
	attachCaptions(blocks, 11)
	if img.Alt != "Figure 1: Sales" {
		t.Errorf("alt = %q", img.Alt)
	}
}
