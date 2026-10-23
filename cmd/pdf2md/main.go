package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/khofesh/misc-app/internal/pdf2md"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type config struct {
	in        string
	out       string
	opts      pdf2md.Options
	images    bool
	overwrite bool
	verbose   bool
}

var errSkipped = errors.New("output exists")

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	cfg, code, done := parseFlags(args, stderr)
	if done {
		return code
	}

	level := zerolog.InfoLevel
	if cfg.verbose {
		level = zerolog.DebugLevel
	}
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: stderr, NoColor: !isTerminal(stderr)}).
		Level(level).With().Timestamp().Logger()

	info, err := os.Stat(cfg.in)
	if err != nil {
		log.Error().Err(err).Str("input", cfg.in).Msg("cannot access input")
		return 1
	}
	if info.IsDir() && cfg.out == "" {
		log.Error().Msg("-o is required when -i is a directory")
		return 1
	}
	if cfg.images && cfg.out == "" {
		log.Error().Msg("-images needs -o (cannot write images next to stdout)")
		return 1
	}

	c, err := pdf2md.New(cfg.opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize")
		if errors.Is(err, pdf2md.ErrMissingTool) {
			return 2
		}
		return 1
	}
	log.Debug().Str("backend", string(c.Backend())).Msg("ready")

	ctx := context.Background()
	if info.IsDir() {
		return processDir(ctx, c, cfg)
	}
	if err := processFile(ctx, c, cfg, cfg.in, cfg.out, stdout); err != nil {
		if errors.Is(err, errSkipped) {
			return 0
		}
		log.Error().Err(err).Str("file", cfg.in).Msg("failed")
		return 1
	}
	return 0
}

func parseFlags(args []string, stderr io.Writer) (*config, int, bool) {
	cfg := &config{}
	fs := flag.NewFlagSet("pdf2md", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&cfg.in, "i", "", "input .pdf file or directory (required)")
	fs.StringVar(&cfg.out, "o", "", "output .md file or directory (empty: stdout)")
	fs.StringVar(&cfg.opts.Pages, "pages", "", "page selection, e.g. 1-3,7,10-")
	fs.StringVar(&cfg.opts.Password, "password", "", "user password for encrypted PDFs")
	backend := fs.String("backend", "auto", "auto, mutool, or native")
	fs.StringVar(&cfg.opts.MutoolPath, "mutool", "", "path to the mutool binary")
	ocr := fs.String("ocr", "auto", "auto, never, or always")
	fs.StringVar(&cfg.opts.Lang, "lang", "eng", "tesseract language(s)")
	fs.BoolVar(&cfg.images, "images", false, "extract embedded images into <name>_images/")
	fs.BoolVar(&cfg.opts.PageBreaks, "page-breaks", false, "insert <!-- page N --> between pages")
	fs.BoolVar(&cfg.opts.KeepHeaders, "keep-headers", false, "keep repeated page headers and footers")
	fs.BoolVar(&cfg.opts.Tables, "tables", true, "detect simple tables")
	fs.BoolVar(&cfg.opts.FrontMatter, "front-matter", false, "write a YAML front matter block")
	fs.IntVar(&cfg.opts.Workers, "workers", runtime.NumCPU(), "pages processed in parallel")
	fs.BoolVar(&cfg.overwrite, "overwrite", false, "overwrite existing output files")
	fs.BoolVar(&cfg.verbose, "v", false, "verbose logging")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil, 0, true
		}
		return nil, 1, true
	}

	cfg.opts.Backend = pdf2md.Backend(*backend)
	switch cfg.opts.Backend {
	case pdf2md.BackendAuto, pdf2md.BackendMutool, pdf2md.BackendNative:
	default:
		fmt.Fprintf(stderr, "invalid -backend %q (want auto, mutool, or native)\n", *backend)
		return nil, 1, true
	}
	cfg.opts.OCR = pdf2md.OCRMode(*ocr)
	switch cfg.opts.OCR {
	case pdf2md.OCRAuto, pdf2md.OCRNever, pdf2md.OCRAlways:
	default:
		fmt.Fprintf(stderr, "invalid -ocr %q (want auto, never, or always)\n", *ocr)
		return nil, 1, true
	}
	if cfg.in == "" {
		fmt.Fprintln(stderr, "-i is required")
		fs.Usage()
		return nil, 1, true
	}
	if cfg.opts.Workers < 1 {
		cfg.opts.Workers = 1
	}
	return cfg, 0, false
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func isPDF(name string) bool {
	return strings.EqualFold(filepath.Ext(name), ".pdf")
}

func processDir(ctx context.Context, c *pdf2md.Converter, cfg *config) int {
	entries, err := os.ReadDir(cfg.in)
	if err != nil {
		log.Error().Err(err).Str("dir", cfg.in).Msg("cannot read directory")
		return 1
	}
	if err := os.MkdirAll(cfg.out, 0o755); err != nil {
		log.Error().Err(err).Str("dir", cfg.out).Msg("cannot create output directory")
		return 1
	}

	var ok, failed, skipped int
	for _, e := range entries {
		if e.IsDir() || !isPDF(e.Name()) {
			continue
		}
		src := filepath.Join(cfg.in, e.Name())
		dst := filepath.Join(cfg.out, strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))+".md")
		switch err := processFile(ctx, c, cfg, src, dst, nil); {
		case errors.Is(err, errSkipped):
			skipped++
		case err != nil:
			failed++
			log.Error().Err(err).Str("file", src).Msg("failed")
		default:
			ok++
		}
	}
	log.Info().Int("ok", ok).Int("failed", failed).Int("skipped", skipped).Msg("done")
	if failed > 0 {
		return 3
	}
	return 0
}

// processFile converts src into dst, or into stdout when dst is empty.
func processFile(ctx context.Context, c *pdf2md.Converter, cfg *config, src, dst string, stdout io.Writer) error {
	if dst != "" && !cfg.overwrite {
		if _, err := os.Stat(dst); err == nil {
			log.Warn().Str("file", dst).Msg("output exists, skipping (use -overwrite)")
			return errSkipped
		}
	}
	if dst != "" {
		if dir := filepath.Dir(dst); dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
		}
	}

	conv := c
	if cfg.images {
		conv = c.WithImageDir(strings.TrimSuffix(dst, filepath.Ext(dst)) + "_images")
	}

	start := time.Now()
	var buf bytes.Buffer
	st, err := conv.Convert(ctx, src, &buf)
	if err != nil {
		return err
	}
	if dst == "" {
		_, err = stdout.Write(buf.Bytes())
	} else {
		err = os.WriteFile(dst, buf.Bytes(), 0o644)
	}
	if err != nil {
		return err
	}
	log.Info().Str("file", src).Int("pages", st.Pages).Int("ocr", st.OCRPages).
		Int("images", st.Images).Int("tables", st.Tables).Dur("elapsed", time.Since(start)).Msg("converted")
	return nil
}
