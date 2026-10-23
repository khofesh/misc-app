package pdf2md

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

const ocrDPI = 300

// needsOCR implements the -ocr auto rule (§6).
func needsOCR(pg *Page) bool {
	if pg.charCount() >= 20 {
		return false
	}
	area := pg.W * pg.H
	for _, img := range pg.Images {
		if area <= 0 || img.BBox.W()*img.BBox.H() >= 0.5*area {
			return true
		}
	}
	return false
}

// ocrPage renders the page, runs tesseract and replaces the page's spans
// with the recognized words.
func (c *Converter) ocrPage(ctx context.Context, path string, pg *Page) error {
	dir, err := os.MkdirTemp("", "pdf2md-ocr")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	img, scale, err := c.pageImage(ctx, path, pg, dir)
	if err != nil {
		return err
	}

	args := []string{img, "-", "-l", c.opts.Lang, "hocr"}
	cmd := exec.CommandContext(ctx, c.tesseract, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("tesseract: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	spans, err := parseHOCR(bytes.NewReader(out), scale)
	if err != nil {
		return err
	}
	pg.Spans = spans
	pg.OCR = true
	pg.Failed = nil
	return nil
}

// pageImage renders the page with mutool, or extracts its largest embedded
// image when mutool is not available. It returns the image path and the
// factor that maps image pixels to page points.
func (c *Converter) pageImage(ctx context.Context, path string, pg *Page, dir string) (string, float64, error) {
	if c.mutool != "" {
		out := filepath.Join(dir, "page.png")
		m := mutoolExtractor{bin: c.mutool, password: c.opts.Password}
		args := m.withPassword("draw", "-q", "-r", strconv.Itoa(ocrDPI), "-o", out, path, strconv.Itoa(pg.Num))
		if _, err := m.run(ctx, args...); err != nil {
			return "", 0, err
		}
		return out, 72.0 / ocrDPI, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	imgs, err := api.ExtractImagesRaw(ctx, f, []string{strconv.Itoa(pg.Num)}, pdfcpuConf(c.opts.Password))
	if err != nil {
		return "", 0, fmt.Errorf("extract page image: %w", err)
	}
	var best *imageData
	for _, m := range imgs {
		for _, img := range m {
			data, err := io.ReadAll(img)
			if err != nil {
				return "", 0, err
			}
			cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
			if err != nil {
				continue
			}
			if best == nil || cfg.Width*cfg.Height > best.w*best.h {
				best = &imageData{w: cfg.Width, h: cfg.Height, ext: img.FileType, data: data}
			}
		}
	}
	if best == nil {
		return "", 0, errors.New("no page image to OCR")
	}
	out := filepath.Join(dir, "page."+best.ext)
	if err := os.WriteFile(out, best.data, 0o644); err != nil {
		return "", 0, err
	}
	scale := 72.0 / ocrDPI
	if pg.W > 0 && best.w > 0 {
		scale = pg.W / float64(best.w)
	}
	return out, scale, nil
}

type imageData struct {
	w, h int
	ext  string
	data []byte
}

var (
	bboxRe     = regexp.MustCompile(`bbox (-?\d+) (-?\d+) (-?\d+) (-?\d+)`)
	baselineRe = regexp.MustCompile(`baseline (-?[\d.]+) (-?[\d.]+)`)
	xsizeRe    = regexp.MustCompile(`x_size ([\d.]+)`)
)

func titleBBox(title string) (Rect, bool) {
	m := bboxRe.FindStringSubmatch(title)
	if m == nil {
		return Rect{}, false
	}
	var v [4]float64
	for i := range v {
		v[i], _ = strconv.ParseFloat(m[i+1], 64)
	}
	return Rect{v[0], v[1], v[2], v[3]}, true
}

func hasClass(e xml.StartElement, class string) bool {
	return strings.Contains(" "+attr(e, "class")+" ", " "+class+" ")
}

// parseHOCR converts tesseract hOCR into spans, one per word, scaled from
// pixels to points.
func parseHOCR(r io.Reader, scale float64) ([]Span, error) {
	d := xml.NewDecoder(r)
	d.Strict = false
	d.AutoClose = xml.HTMLAutoClose
	d.Entity = xml.HTMLEntity

	var (
		spans                []Span
		line                 Rect
		baseM, baseC, lineSz float64
		inWord               bool
		depth, wordDepth     int
		word                 Rect
		text                 strings.Builder
		bold, italic         bool
	)
	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse hocr: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			title := attr(t, "title")
			switch {
			case hasClass(t, "ocr_line") || hasClass(t, "ocr_caption") || hasClass(t, "ocr_header") || hasClass(t, "ocr_textfloat"):
				line, _ = titleBBox(title)
				baseM, baseC, lineSz = 0, 0, 0
				if m := baselineRe.FindStringSubmatch(title); m != nil {
					baseM, _ = strconv.ParseFloat(m[1], 64)
					baseC, _ = strconv.ParseFloat(m[2], 64)
				}
				if m := xsizeRe.FindStringSubmatch(title); m != nil {
					lineSz, _ = strconv.ParseFloat(m[1], 64)
				}
			case hasClass(t, "ocrx_word"):
				word, _ = titleBBox(title)
				inWord, wordDepth = true, depth
				text.Reset()
				bold, italic = false, false
			case inWord && (t.Name.Local == "strong" || t.Name.Local == "b"):
				bold = true
			case inWord && (t.Name.Local == "em" || t.Name.Local == "i"):
				italic = true
			}
		case xml.CharData:
			if inWord {
				text.Write(t)
			}
		case xml.EndElement:
			if inWord && depth == wordDepth {
				inWord = false
				if w := strings.TrimSpace(text.String()); w != "" {
					base := line.Y1 + baseC + baseM*((word.X0+word.X1)/2-line.X0)
					if line == (Rect{}) || base < word.Y0 {
						base = word.Y1
					}
					size := lineSz
					if size == 0 {
						size = base - word.Y0
					}
					spans = append(spans, Span{
						Text: w, X: word.X0 * scale, Y: base * scale,
						W: word.W() * scale, H: max(base-word.Y0, 1) * scale,
						Size: size * scale, Bold: bold, Italic: italic,
					})
				}
			}
			depth--
		}
	}
	return spans, nil
}
