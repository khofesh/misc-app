package htmltag

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func HtmlToText(htmlContent string) (string, error) {
	// parse html
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	// remove script and style elements
	doc.Find("script, style").Remove()

	var textBuilder strings.Builder

	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		tagName := goquery.NodeName(s)
		text := strings.TrimSpace(s.Text())

		// skip empty text
		if text == "" {
			return
		}

		// handle different html elements
		switch tagName {
		case "h1", "h2", "h3", "h4", "h5", "h6":
			textBuilder.WriteString("\n\n")
			textBuilder.WriteString(strings.ToUpper(text))
			textBuilder.WriteString("\n")
			textBuilder.WriteString(strings.Repeat("=", len(text)))
			textBuilder.WriteString("\n")
		case "p":
			// paragraphs with double line break
			textBuilder.WriteString("\n\n")
			textBuilder.WriteString(text)
		case "br":
			// line breaks
			textBuilder.WriteString("\n")
		case "li":
			// list items with bullet points
			textBuilder.WriteString("\n• ")
			textBuilder.WriteString(text)
		case "a":
			// links with url if available
			if href, exists := s.Attr("href"); exists {
				textBuilder.WriteString(fmt.Sprintf("%s (%s)", text, href))
			} else {
				textBuilder.WriteString(text)
			}
		case "div", "span":
			// generic containers - just add the text with spacing
			if textBuilder.Len() > 0 && !strings.HasSuffix(textBuilder.String(), "\n") {
				textBuilder.WriteString(" ")
			}
			textBuilder.WriteString(text)
		default:
			// for other elements, just add text with appropriate spacing
			if textBuilder.Len() > 0 && !strings.HasSuffix(textBuilder.String(), "\n") && !strings.HasSuffix(textBuilder.String(), " ") {
				textBuilder.WriteString(" ")
			}
			textBuilder.WriteString(text)
		}
	})

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

	return strings.Join(cleanLines, "\n"), nil
}
