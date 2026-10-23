package pdf2md

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	_ "golang.org/x/image/tiff"
)

const (
	minImagePx    = 32
	minImageBytes = 1024
	minImagePt    = 24 // positions smaller than this are treated as decorations
)

// extractImages writes the embedded images of the selected pages into dir and
// returns their link paths (relative to the Markdown file) per page.
func extractImages(ctx context.Context, path, password, dir string, pages []int) (map[int][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sel := make([]string, len(pages))
	for i, p := range pages {
		sel[i] = strconv.Itoa(p)
	}
	raw, err := api.ExtractImagesRaw(ctx, f, sel, pdfcpuConf(password))
	if err != nil {
		return nil, fmt.Errorf("extract images: %w", err)
	}

	type extracted struct {
		obj  int
		ext  string
		data []byte
	}
	byPage := map[int][]extracted{}
	for _, m := range raw {
		for _, img := range m {
			if img.Thumb {
				continue
			}
			// pdfcpu leaves size and dimensions unset for raw extraction
			data, err := io.ReadAll(img)
			if err != nil {
				return nil, fmt.Errorf("read image %d: %w", img.ObjNr, err)
			}
			if len(data) < minImageBytes {
				continue
			}
			if cfg, _, err := image.DecodeConfig(bytes.NewReader(data)); err == nil && (cfg.Width < minImagePx || cfg.Height < minImagePx) {
				continue
			}
			byPage[img.PageNr] = append(byPage[img.PageNr], extracted{img.ObjNr, img.FileType, data})
		}
	}
	if len(byPage) == 0 {
		return nil, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	out := map[int][]string{}
	for pg, imgs := range byPage {
		sort.Slice(imgs, func(i, j int) bool { return imgs[i].obj < imgs[j].obj })
		for n, img := range imgs {
			name := fmt.Sprintf("p%d_%d.%s", pg, n+1, img.ext)
			if err := os.WriteFile(filepath.Join(dir, name), img.data, 0o644); err != nil {
				return nil, err
			}
			out[pg] = append(out[pg], filepath.Base(dir)+"/"+name)
		}
	}
	return out, nil
}

// placeImages matches extracted image files to image positions on the page
// in order. Files without a position go to the end of the page.
func placeImages(pg *Page, files []string) {
	var placed []ImageRef
	for _, img := range pg.Images {
		if img.BBox.W() >= minImagePt && img.BBox.H() >= minImagePt {
			placed = append(placed, img)
		}
	}
	for i, f := range files {
		if i < len(placed) {
			placed[i].Path = f
			continue
		}
		y := pg.H + float64(i)
		placed = append(placed, ImageRef{BBox: Rect{0, y, pg.W, y + 1}, Path: f})
	}
	pg.Images = placed
}
