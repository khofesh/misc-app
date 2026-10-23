package pdf2md

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type OCRMode string

const (
	OCRAuto   OCRMode = "auto"
	OCRNever  OCRMode = "never"
	OCRAlways OCRMode = "always"
)

type Options struct {
	Pages       string // page selection like "1-3,7,10-"; empty means all
	Password    string
	Backend     Backend // "auto", "mutool", "native"
	MutoolPath  string
	OCR         OCRMode
	Lang        string
	ImageDir    string // empty disables image extraction
	PageBreaks  bool
	KeepHeaders bool
	Tables      bool
	FrontMatter bool
	Workers     int
}

type Stats struct {
	Pages, OCRPages, Images, Tables int
}

type Converter struct {
	opts      Options
	ext       Extractor
	backend   Backend
	mutool    string
	tesseract string
	ranges    [][2]int
	ocrWarn   *sync.Once
}

var errNoText = errors.New("no text")

func New(opts Options) (*Converter, error) {
	if opts.Workers <= 0 {
		opts.Workers = runtime.NumCPU()
	}
	if opts.Lang == "" {
		opts.Lang = "eng"
	}
	switch opts.OCR {
	case "":
		opts.OCR = OCRAuto
	case OCRAuto, OCRNever, OCRAlways:
	default:
		return nil, fmt.Errorf("invalid ocr mode %q", opts.OCR)
	}
	ranges, err := parsePageRanges(opts.Pages)
	if err != nil {
		return nil, err
	}

	c := &Converter{opts: opts, ranges: ranges, mutool: FindMutool(opts.MutoolPath), ocrWarn: &sync.Once{}}
	c.ext, c.backend, err = newExtractor(opts.Backend, c.mutool, opts.Password)
	if err != nil {
		return nil, err
	}
	if c.backend == BackendNative && c.mutool == "" && opts.Backend != BackendNative {
		log.Warn().Msg("mutool not found, using the native backend (install mupdf for better results)")
	}
	if c.backend == BackendNative {
		c.mutool = ""
	}

	c.tesseract, _ = exec.LookPath("tesseract")
	if opts.OCR == OCRAlways && c.tesseract == "" {
		return nil, fmt.Errorf("-ocr always: %w: tesseract not found", ErrMissingTool)
	}
	return c, nil
}

// WithImageDir returns a copy of c that extracts images into dir.
func (c *Converter) WithImageDir(dir string) *Converter {
	cc := *c
	cc.opts.ImageDir = dir
	return &cc
}

// Backend returns the extraction backend in use.
func (c *Converter) Backend() Backend { return c.backend }

// parsePageRanges parses "1-3,7,10-". An open end is stored as 0.
func parsePageRanges(spec string) ([][2]int, error) {
	if strings.TrimSpace(spec) == "" {
		return nil, nil
	}
	var out [][2]int
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		from, to, isRange := strings.Cut(part, "-")
		a, err := strconv.Atoi(strings.TrimSpace(from))
		if err != nil || a < 1 {
			return nil, fmt.Errorf("invalid page selection %q", part)
		}
		b := a
		if isRange {
			b = 0
			if strings.TrimSpace(to) != "" {
				b, err = strconv.Atoi(strings.TrimSpace(to))
				if err != nil || b < a {
					return nil, fmt.Errorf("invalid page selection %q", part)
				}
			}
		}
		out = append(out, [2]int{a, b})
	}
	return out, nil
}

// selectPages expands the ranges for a document of n pages, sorted and
// without duplicates.
func selectPages(ranges [][2]int, n int) []int {
	if len(ranges) == 0 {
		ranges = [][2]int{{1, 0}}
	}
	seen := make([]bool, n+1)
	for _, r := range ranges {
		to := r[1]
		if to == 0 || to > n {
			to = n
		}
		for p := r[0]; p <= to; p++ {
			seen[p] = true
		}
	}
	var out []int
	for p := 1; p <= n; p++ {
		if seen[p] {
			out = append(out, p)
		}
	}
	return out
}

func parallel(n, workers int, fn func(i int)) {
	ch := make(chan int)
	var wg sync.WaitGroup
	for range max(1, min(workers, n)) {
		wg.Go(func() {
			for i := range ch {
				fn(i)
			}
		})
	}
	for i := range n {
		ch <- i
	}
	close(ch)
	wg.Wait()
}

// extractAll extracts pages in contiguous chunks, one per worker. A chunk
// that fails marks only its own pages as failed.
func (c *Converter) extractAll(ctx context.Context, path string, pages []int) []Page {
	nChunks := max(1, min(c.opts.Workers, len(pages)))
	size := (len(pages) + nChunks - 1) / nChunks
	var chunks [][]int
	for i := 0; i < len(pages); i += size {
		chunks = append(chunks, pages[i:min(i+size, len(pages))])
	}

	byNum := make(map[int]Page, len(pages))
	var mu sync.Mutex
	parallel(len(chunks), c.opts.Workers, func(i int) {
		got, err := c.ext.Extract(ctx, path, chunks[i])
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			log.Warn().Err(err).Str("file", path).Str("pages", formatPages(chunks[i])).Msg("text extraction failed")
		}
		for _, pg := range got {
			byNum[pg.Num] = pg
		}
		for _, p := range chunks[i] {
			if _, ok := byNum[p]; !ok {
				byNum[p] = Page{Num: p, Failed: fmt.Errorf("page %d: %w", p, err)}
			}
		}
	})

	out := make([]Page, len(pages))
	for i, p := range pages {
		out[i] = byNum[p]
	}
	return out
}

func (c *Converter) wantOCR(pg *Page) bool {
	switch c.opts.OCR {
	case OCRAlways:
		return true
	case OCRNever:
		return false
	}
	if pg.Failed != nil {
		return true
	}
	if c.backend == BackendNative {
		return pg.charCount() < 20
	}
	return needsOCR(pg)
}

func (c *Converter) runOCR(ctx context.Context, path string, pages []Page) {
	var todo []int
	for i := range pages {
		if c.wantOCR(&pages[i]) {
			todo = append(todo, i)
		}
	}
	if len(todo) == 0 {
		return
	}
	if c.tesseract == "" {
		c.ocrWarn.Do(func() {
			log.Warn().Msg("tesseract not found, skipping OCR for pages without text")
		})
		return
	}
	parallel(len(todo), c.opts.Workers, func(i int) {
		pg := &pages[todo[i]]
		start := time.Now()
		err := c.ocrPage(ctx, path, pg)
		if err != nil {
			log.Warn().Err(err).Str("file", path).Int("page", pg.Num).Msg("ocr failed")
			return
		}
		log.Debug().Str("file", path).Int("page", pg.Num).Dur("elapsed", time.Since(start)).Msg("ocr")
	})
}

// Convert converts the PDF at path to Markdown written to w.
func (c *Converter) Convert(ctx context.Context, path string, w io.Writer) (Stats, error) {
	var st Stats
	n, err := c.ext.PageCount(ctx, path)
	if err != nil {
		return st, fmt.Errorf("%s: %w", path, err)
	}
	sel := selectPages(c.ranges, n)
	if len(sel) == 0 {
		return st, fmt.Errorf("%s: no pages selected (document has %d)", path, n)
	}

	start := time.Now()
	pages := c.extractAll(ctx, path, sel)
	for i := range pages {
		pages[i].Spans = normalizeSpans(pages[i].Spans)
	}
	log.Debug().Str("file", path).Int("pages", len(pages)).Dur("elapsed", time.Since(start)).Msg("extracted")

	m, err := loadMeta(ctx, path, c.opts.Password, sel)
	if err != nil {
		log.Debug().Err(err).Str("file", path).Msg("metadata unavailable")
	}

	c.runOCR(ctx, path, pages)

	var files map[int][]string
	if c.opts.ImageDir != "" {
		files, err = extractImages(ctx, path, c.opts.Password, c.opts.ImageDir, sel)
		if err != nil {
			log.Warn().Err(err).Str("file", path).Msg("image extraction failed")
		}
	}

	for i := range pages {
		pg := &pages[i]
		if pg.OCR {
			st.OCRPages++
		}
		pg.Spans = applyLinks(pg.Spans, m.Links[pg.Num])
		if c.opts.ImageDir != "" {
			placeImages(pg, files[pg.Num])
			st.Images += len(files[pg.Num])
		} else {
			pg.Images = nil
		}
		pg.frags = append(buildFragments(pg.Spans), imageFrags(pg.Images)...)
		if pg.Failed == nil && len(pg.frags) == 0 {
			pg.Failed = errNoText
			log.Warn().Str("file", path).Int("page", pg.Num).Msg("page has no text")
		} else if pg.Failed != nil && !pg.OCR {
			log.Warn().Err(pg.Failed).Str("file", path).Int("page", pg.Num).Msg("page skipped")
		}
	}

	if !c.opts.KeepHeaders {
		removeHeadersFooters(pages)
	}
	body := bodySize(pages)
	log.Debug().Str("file", path).Float64("body_size", body).Msg("layout")

	for i := range pages {
		pages[i].Flows = orderFlows(pages[i].frags, body)
	}
	lead := leadingRatio(pages)
	for i := range pages {
		for _, flow := range pages[i].Flows {
			pages[i].Blocks = append(pages[i].Blocks, buildBlocks(flow, lead, c.opts.Tables)...)
		}
	}

	cls := newClassifier(pages, body, m.Outline)
	for i := range pages {
		cls.classify(pages[i].Blocks)
		attachCaptions(pages[i].Blocks, body)
		for _, b := range pages[i].Blocks {
			if b.Kind == KindTable {
				st.Tables++
			}
		}
	}
	mergeAcrossPages(pages)

	var fm *FrontMatter
	if c.opts.FrontMatter {
		fm = &FrontMatter{Title: m.Title, Author: m.Author, Pages: n, Source: filepath.Base(path)}
		if fm.Title == "" {
			fm.Title = firstHeading(pages)
		}
	}
	st.Pages = len(pages)
	return st, renderDoc(w, pages, fm, c.opts.PageBreaks)
}

func firstHeading(pages []Page) string {
	for _, pg := range pages {
		for _, b := range pg.Blocks {
			if b.Kind == KindHeading && b.Level == 1 {
				return collapseSpace(b.Text())
			}
		}
	}
	return ""
}
