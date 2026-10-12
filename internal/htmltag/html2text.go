package htmltag

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
)

// blockTags start on a new line and end with a line break.
var blockTags = map[string]bool{
	"div": true, "section": true, "article": true, "header": true, "footer": true,
	"main": true, "nav": true, "aside": true, "blockquote": true, "pre": true,
	"ul": true, "ol": true, "table": true, "tr": true, "form": true, "title": true,
}

func HtmlToText(htmlContent string) (string, error) {
	// parse html
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	// remove script and style elements
	doc.Find("script, style").Remove()

	var textBuilder strings.Builder
	writeChildren(&textBuilder, doc.Selection)

	result := textBuilder.String()

	// remove excessive whitespace
	lines := strings.Split(result, "\n")

	var cleanLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" || (len(cleanLines) > 0 && cleanLines[len(cleanLines)-1] != "") {
			cleanLines = append(cleanLines, trimmed)
		}
	}

	// drop trailing blank lines
	for len(cleanLines) > 0 && cleanLines[len(cleanLines)-1] == "" {
		cleanLines = cleanLines[:len(cleanLines)-1]
	}

	return strings.Join(cleanLines, "\n"), nil
}

// writeChildren walks the direct children of s, so every text node is
// written exactly once.
func writeChildren(b *strings.Builder, s *goquery.Selection) {
	s.Contents().Each(func(i int, c *goquery.Selection) {
		writeNode(b, c)
	})
}

func writeNode(b *strings.Builder, s *goquery.Selection) {
	tagName := goquery.NodeName(s)

	switch tagName {
	case "#text":
		writeInline(b, s.Text())
	case "#comment":
		// skip
	case "h1", "h2", "h3", "h4", "h5", "h6":
		text := strings.TrimSpace(collapseSpace(s.Text()))
		if text == "" {
			return
		}
		b.WriteString("\n\n")
		b.WriteString(strings.ToUpper(text))
		b.WriteString("\n")
		b.WriteString(strings.Repeat("=", utf8.RuneCountInString(text)))
		b.WriteString("\n\n")
	case "p":
		// paragraphs with double line break
		b.WriteString("\n\n")
		writeChildren(b, s)
		b.WriteString("\n\n")
	case "br":
		// line breaks
		b.WriteString("\n")
	case "li":
		// list items with bullet points
		ensureNewline(b)
		b.WriteString("• ")
		writeChildren(b, s)
		ensureNewline(b)
	case "a":
		// links with url if available
		writeChildren(b, s)
		if href, exists := s.Attr("href"); exists {
			writeInline(b, fmt.Sprintf(" (%s)", href))
		}
	case "td", "th":
		// table cells separated by spaces
		writeInline(b, " ")
		writeChildren(b, s)
		writeInline(b, " ")
	default:
		if blockTags[tagName] {
			ensureNewline(b)
			writeChildren(b, s)
			ensureNewline(b)
			return
		}
		// inline elements (span, strong, em, ...) just contribute their text
		writeChildren(b, s)
	}
}

// writeInline writes text with HTML whitespace collapsed, avoiding
// double spaces and spaces at the start of a line.
func writeInline(b *strings.Builder, text string) {
	text = collapseSpace(text)
	if strings.HasPrefix(text, " ") && atLineStartOrSpace(b) {
		text = text[1:]
	}
	b.WriteString(text)
}

// collapseSpace replaces runs of HTML whitespace with a single space,
// keeping one leading or trailing space if the text had any.
// Non-breaking spaces are kept, like a browser does.
func collapseSpace(text string) string {
	words := strings.FieldsFunc(text, isHTMLSpace)
	if len(words) == 0 {
		if text == "" {
			return ""
		}
		return " "
	}

	out := strings.Join(words, " ")
	if isHTMLSpace(rune(text[0])) {
		out = " " + out
	}
	if isHTMLSpace(rune(text[len(text)-1])) {
		out += " "
	}
	return out
}

func isHTMLSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f'
}

func atLineStartOrSpace(b *strings.Builder) bool {
	s := b.String()
	return s == "" || strings.HasSuffix(s, "\n") || strings.HasSuffix(s, " ")
}

func ensureNewline(b *strings.Builder) {
	if b.Len() > 0 && !strings.HasSuffix(b.String(), "\n") {
		b.WriteString("\n")
	}
}
