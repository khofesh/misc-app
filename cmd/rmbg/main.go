//go:build linux

package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/khofesh/misc-app/internal/rmbg"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type config struct {
	in        string
	out       string
	model     string
	device    rmbg.Device
	gpuID     int
	ortLib    string
	workers   int
	threads   int
	maskOnly  bool
	bg        string
	bgColor   *color.NRGBA
	threshold int
	overwrite bool
	verbose   bool
}

func main() {
	os.Exit(run(os.Args[1:], os.Stderr))
}

func run(args []string, stderr io.Writer) int {
	cfg, code, done := parseFlags(args, stderr)
	if done {
		return code
	}

	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: stderr}).With().Timestamp().Logger()

	info, err := os.Stat(cfg.in)
	if err != nil {
		log.Error().Err(err).Str("input", cfg.in).Msg("cannot access input")
		return 1
	}

	r, err := rmbg.New(rmbg.Options{
		Model:      cfg.model,
		Device:     cfg.device,
		GPUID:      cfg.gpuID,
		ORTLibPath: cfg.ortLib,
		Threads:    cfg.threads,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize")
		if errors.Is(err, rmbg.ErrInvalidModel) {
			return 1
		}
		return 2
	}
	defer r.Close()

	l := log.Info().Str("device", string(r.Device())).Int("size", r.Size()).Str("model", cfg.model)
	if r.Device() == rmbg.DeviceCUDA {
		l = l.Int("gpu", cfg.gpuID)
	}
	l.Msg("loaded model")

	if info.IsDir() {
		return processDir(r, cfg)
	}
	if dir := filepath.Dir(cfg.out); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Error().Err(err).Str("dir", dir).Msg("cannot create output directory")
			return 1
		}
	}
	if err := processFile(r, cfg, cfg.in, cfg.out); err != nil {
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
	fs := flag.NewFlagSet("rmbg", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&cfg.in, "i", "", "input file or directory (required)")
	fs.StringVar(&cfg.out, "o", "", "output file or directory (required)")
	fs.StringVar(&cfg.model, "model", "u2net", "model name or path to a .onnx file")
	device := fs.String("device", "auto", "auto, cpu, or cuda")
	fs.IntVar(&cfg.gpuID, "gpu-id", 0, "CUDA device index")
	fs.StringVar(&cfg.ortLib, "ort-lib", "", "path to libonnxruntime.so")
	fs.IntVar(&cfg.workers, "workers", 1, "images pre/post-processed in parallel")
	fs.IntVar(&cfg.threads, "threads", 0, "ONNX Runtime intra-op threads (0 = auto)")
	fs.BoolVar(&cfg.maskOnly, "mask-only", false, "write the alpha mask instead of the image")
	fs.StringVar(&cfg.bg, "bg", "", "fill the background with a hex color")
	fs.IntVar(&cfg.threshold, "alpha-threshold", 0, "binary mask level (1-254)")
	fs.BoolVar(&cfg.overwrite, "overwrite", false, "overwrite existing output files")
	fs.BoolVar(&cfg.verbose, "v", false, "verbose logging")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil, 0, true
		}
		return nil, 1, true
	}

	cfg.device = rmbg.Device(*device)
	switch cfg.device {
	case rmbg.DeviceAuto, rmbg.DeviceCPU, rmbg.DeviceCUDA:
	default:
		fmt.Fprintf(stderr, "invalid -device %q (want auto, cpu, or cuda)\n", *device)
		return nil, 1, true
	}

	if cfg.in == "" || cfg.out == "" {
		fmt.Fprintln(stderr, "-i and -o are required")
		fs.Usage()
		return nil, 1, true
	}
	if cfg.workers < 1 {
		cfg.workers = 1
	}
	if cfg.threshold < 0 || cfg.threshold > 254 {
		fmt.Fprintf(stderr, "invalid -alpha-threshold %d (want 0-254)\n", cfg.threshold)
		return nil, 1, true
	}
	if cfg.bg != "" {
		c, err := rmbg.ParseHexColor(cfg.bg)
		if err != nil {
			fmt.Fprintf(stderr, "invalid -bg: %v\n", err)
			return nil, 1, true
		}
		cfg.bgColor = &c
	}
	return cfg, 0, false
}

func processDir(r *rmbg.Remover, cfg *config) int {
	entries, err := os.ReadDir(cfg.in)
	if err != nil {
		log.Error().Err(err).Str("dir", cfg.in).Msg("cannot read directory")
		return 1
	}
	if err := os.MkdirAll(cfg.out, 0o755); err != nil {
		log.Error().Err(err).Str("dir", cfg.out).Msg("cannot create output directory")
		return 1
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() || !isSupportedImage(e.Name()) {
			continue
		}
		files = append(files, e.Name())
	}

	type result struct {
		name    string
		err     error
		skipped bool
	}
	jobs := make(chan string)
	results := make(chan result)
	var wg sync.WaitGroup

	for i := 0; i < cfg.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for name := range jobs {
				src := filepath.Join(cfg.in, name)
				dst := filepath.Join(cfg.out, outputBase(name)+".png")
				err := processFile(r, cfg, src, dst)
				results <- result{name: name, err: err, skipped: errors.Is(err, errSkipped)}
			}
		}()
	}

	go func() {
		for _, f := range files {
			jobs <- f
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	var ok, failed, skipped int
	for res := range results {
		switch {
		case res.skipped:
			skipped++
		case res.err != nil:
			failed++
			log.Error().Err(res.err).Str("file", res.name).Msg("failed")
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

var errSkipped = errors.New("output exists")

func processFile(r *rmbg.Remover, cfg *config, src, dst string) error {
	if !cfg.overwrite {
		if _, err := os.Stat(dst); err == nil {
			log.Warn().Str("file", dst).Msg("output exists, skipping")
			return errSkipped
		}
	}

	start := time.Now()
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	img, err := rmbg.Decode(data)
	if err != nil {
		return err
	}
	decodeDur := time.Since(start)

	preStart := time.Now()
	tensor, err := r.Preprocess(img)
	if err != nil {
		return err
	}
	preDur := time.Since(preStart)

	infStart := time.Now()
	raw, err := r.Infer(tensor)
	if err != nil {
		return err
	}
	infDur := time.Since(infStart)

	postStart := time.Now()
	mask := r.Postprocess(raw, img.Bounds().Dx(), img.Bounds().Dy())
	rmbg.Threshold(mask, cfg.threshold)
	if err := encodeOutput(dst, img, mask, cfg); err != nil {
		return err
	}
	postDur := time.Since(postStart)

	ev := log.Info().Str("file", src).Str("out", dst)
	if cfg.verbose {
		ev = ev.
			Dur("decode", decodeDur).
			Dur("preprocess", preDur).
			Dur("inference", infDur).
			Dur("postprocess", postDur).
			Dur("total", time.Since(start))
	}
	ev.Msg("processed")
	return nil
}

func encodeOutput(dst string, img image.Image, mask *image.Gray, cfg *config) error {
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()

	if cfg.maskOnly {
		return png.Encode(f, mask)
	}
	if cfg.bgColor != nil {
		out := rmbg.Blend(img, mask, *cfg.bgColor)
		if isJPEG(dst) {
			return jpeg.Encode(f, out, &jpeg.Options{Quality: 95})
		}
		return png.Encode(f, out)
	}
	if isJPEG(dst) {
		return fmt.Errorf("cannot write a transparent image to %s; use -bg or a .png output", dst)
	}
	return png.Encode(f, rmbg.ApplyAlpha(img, mask))
}

// outputBase returns the file name without its extension.
func outputBase(name string) string {
	return strings.TrimSuffix(name, filepath.Ext(name))
}

func isSupportedImage(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".webp":
		return true
	default:
		return false
	}
}

func isJPEG(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg":
		return true
	default:
		return false
	}
}
