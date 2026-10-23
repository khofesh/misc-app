package pdf2md

import "testing"

func TestRemoveHeadersFooters(t *testing.T) {
	page := func(n string) Page {
		return Page{W: 612, H: 792, frags: buildFragments([]Span{
			sp("Annual Report 2024", 72, 30, 9),
			sp("Body text on page "+n, 72, 300, 11),
			sp(n, 300, 770, 9),
		})}
	}
	texts := func(pg Page) []string {
		var out []string
		for _, f := range pg.frags {
			out = append(out, f.Text())
		}
		return out
	}

	t.Run("repeated", func(t *testing.T) {
		pages := []Page{page("1"), page("2"), page("3")}
		removeHeadersFooters(pages)
		for i, pg := range pages {
			if got := texts(pg); len(got) != 1 || got[0] != "Body text on page "+string(rune('1'+i)) {
				t.Errorf("page %d = %q", i+1, got)
			}
		}
	})

	t.Run("single page keeps header, drops page number", func(t *testing.T) {
		pages := []Page{page("1")}
		removeHeadersFooters(pages)
		if got := texts(pages[0]); !equalStrings(got, []string{"Annual Report 2024", "Body text on page 1"}) {
			t.Errorf("frags = %q", got)
		}
	})
}

func TestJoinLines(t *testing.T) {
	tests := []struct {
		in   []string
		want string
	}{
		{[]string{"a hyphen-", "ated word"}, "a hyphenated word"},
		{[]string{"well-", "Known"}, "well- Known"},
		{[]string{"plain", "join "}, "plain join"},
		{[]string{"2-", "3"}, "2- 3"},
	}
	for _, tt := range tests {
		if got := joinLines(tt.in); got != tt.want {
			t.Errorf("joinLines(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNormalizeText(t *testing.T) {
	if got := normalizeText("ﬁne\u00adtune\u200b\u00a0e\u0301"); got != "finetune é" {
		t.Errorf("normalizeText = %q", got)
	}
}

func TestMergeAcrossPages(t *testing.T) {
	tests := []struct {
		name      string
		end, next string
		merged    bool
	}{
		{"continues", "the sentence goes on", "and ends here.", true},
		{"ended", "the sentence ends.", "and a new one.", false},
		{"capital", "no punctuation", "New paragraph", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pages := []Page{
				{Blocks: []Block{para(textLines(72, 700, 11, tt.end)...)}},
				{Blocks: []Block{para(textLines(72, 100, 11, tt.next)...)}},
			}
			mergeAcrossPages(pages)
			if got := len(pages[1].Blocks) == 0; got != tt.merged {
				t.Errorf("merged = %v, want %v", got, tt.merged)
			}
		})
	}
}
