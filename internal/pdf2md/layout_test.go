package pdf2md

import (
	"strings"
	"testing"
)

func TestBuildFragments(t *testing.T) {
	hello := sp("Hello", 72, 100, 10)
	world := sp("world", hello.X+hello.W+3, 100, 10)
	sup := sp("2", world.X+world.W, 97, 6)
	right := sp("Right", 400, 100, 10)
	next := sp("Next", 72, 114, 10)

	frags := buildFragments([]Span{next, right, sup, world, hello})
	var got []string
	for _, f := range frags {
		got = append(got, f.Text())
	}
	want := []string{"Hello world2", "Right", "Next"}
	if !equalStrings(got, want) {
		t.Fatalf("fragments = %q, want %q", got, want)
	}
	if frags[0].row != frags[1].row || frags[1].row == frags[2].row {
		t.Errorf("rows = %d %d %d, want first two equal", frags[0].row, frags[1].row, frags[2].row)
	}
}

func twoColumnFrags() []frag {
	spans := []Span{sp("A Title That Spans The Page Width", 72, 60, 20)}
	for i := range 8 {
		y := 100 + float64(i)*12
		spans = append(spans,
			sp("left column words that fill a line", 72, y, 10),
			sp("right column words that fill a line", 320, y, 10))
	}
	return buildFragments(spans)
}

func TestOrderFlows(t *testing.T) {
	t.Run("two columns", func(t *testing.T) {
		flows := orderFlows(twoColumnFrags(), 10)
		if len(flows) != 3 {
			t.Fatalf("flows = %d, want 3", len(flows))
		}
		if !strings.HasPrefix(flows[0][0].Text(), "A Title") {
			t.Errorf("first flow = %q", flows[0][0].Text())
		}
		for i, prefix := range []string{"left", "right"} {
			f := flows[i+1]
			if len(f) != 8 || !strings.HasPrefix(f[0].Text(), prefix) || !strings.HasPrefix(f[7].Text(), prefix) {
				t.Errorf("flow %d = %d lines starting %q", i+1, len(f), f[0].Text())
			}
		}
	})

	t.Run("single column", func(t *testing.T) {
		var spans []Span
		for i := range 10 {
			spans = append(spans, sp("one column of body text across the page", 72, 100+float64(i)*14, 11))
		}
		if flows := orderFlows(buildFragments(spans), 11); len(flows) != 1 || len(flows[0]) != 10 {
			t.Errorf("flows = %d", len(flows))
		}
	})

	t.Run("narrow number column is not a column", func(t *testing.T) {
		var spans []Span
		for i := range 10 {
			y := 100 + float64(i)*14
			spans = append(spans, sp("1.1 Section title . . . . . . . . . . . . .", 72, y, 11), sp("12", 520, y, 11))
		}
		if g := detectColumns(buildFragments(spans), 11); g != nil {
			t.Errorf("gutters = %v, want none", g)
		}
	})
}

func TestBuildBlocks(t *testing.T) {
	body := "a line of body text that reaches all the way to the right margin"
	tests := []struct {
		name  string
		lines []Line
		want  []string
		kinds []BlockKind
	}{
		{
			name:  "paragraph gap",
			lines: append(textLines(72, 100, 11, body, "ends here."), textLines(72, 140, 11, body)...),
			want:  []string{body + " ends here.", body},
		},
		{
			name:  "size change",
			lines: append(textLines(72, 80, 16, "Heading"), textLines(72, 100, 11, body)...),
			want:  []string{"Heading", body},
		},
		{
			name:  "first line indent",
			lines: append(textLines(90, 100, 11, body), textLines(72, 114, 11, body)...),
			want:  []string{body + " " + body},
		},
		{
			name:  "short line ending a sentence",
			lines: textLines(72, 100, 11, body, "short end.", body),
			want:  []string{body + " short end.", body},
		},
		{
			name: "list items with hanging indent",
			lines: append(append(textLines(72, 100, 11, "• first item "+body),
				textLines(82, 114, 11, "continued")...),
				textLines(72, 128, 11, "• second item")...),
			want: []string{"• first item " + body + " continued", "• second item"},
		},
		{
			name: "code keeps indentation lines together",
			lines: []Line{
				newLine([]Span{mono(sp("func f() {", 72, 100, 10))}),
				newLine([]Span{mono(sp("return", 96, 112, 10))}),
				newLine([]Span{mono(sp("}", 72, 124, 10))}),
			},
			want: []string{"func f() { return }"},
		},
		{
			name: "table",
			lines: []Line{
				newLine([]Span{sp("Item", 72, 100, 11), sp("Price", 300, 100, 11)}),
				newLine([]Span{sp("Milk", 72, 116, 11), sp("1.20", 300, 116, 11)}),
				newLine([]Span{sp("Tea", 72, 132, 11), sp("3.00", 300, 132, 11)}),
			},
			want:  []string{"Item Price Milk 1.20 Tea 3.00"},
			kinds: []BlockKind{KindTable},
		},
		{
			name: "two aligned lines are not a table",
			lines: []Line{
				newLine([]Span{sp("Item", 72, 100, 11), sp("Price", 300, 100, 11)}),
				newLine([]Span{sp("Milk", 72, 116, 11), sp("1.20", 300, 116, 11)}),
			},
			want: []string{"Item Price Milk 1.20"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildBlocks(tt.lines, 1.27, true)
			if texts := blockTexts(got); !equalStrings(texts, tt.want) {
				t.Fatalf("blocks = %q, want %q", texts, tt.want)
			}
			for i, k := range tt.kinds {
				if got[i].Kind != k {
					t.Errorf("block %d kind = %v, want %v", i, got[i].Kind, k)
				}
			}
		})
	}
}

func TestTableRows(t *testing.T) {
	lines := []Line{
		newLine([]Span{sp("Item", 72, 100, 11), sp("Unit", 200, 100, 11), sp("Price", 300, 100, 11)}),
		newLine([]Span{sp("Coffee beans", 72, 116, 11), sp("kg", 200, 116, 11), sp("24.50", 300, 116, 11)}),
		newLine([]Span{sp("Milk", 72, 132, 11), sp("liter", 200, 132, 11), sp("1.20", 300, 132, 11)}),
	}
	b := buildBlocks(lines, 1.2, true)
	if len(b) != 1 || b[0].Kind != KindTable {
		t.Fatalf("blocks = %+v", b)
	}
	if got := strings.Join(b[0].Rows[1], "|"); got != "Coffee beans|kg|24.50" {
		t.Errorf("row 1 = %q", got)
	}
	if b := buildBlocks(lines, 1.2, false); b[0].Kind == KindTable {
		t.Error("table detected with tables disabled")
	}
}
