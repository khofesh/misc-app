//go:build (integration || cuda) && linux

package rmbg

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

// ortLibPath finds libonnxruntime.so for tests: the ONNXRUNTIME_LIB environment
// variable, or ./lib relative to the repo root.
func ortLibPath(tb testing.TB) string {
	tb.Helper()
	if lib := os.Getenv("ONNXRUNTIME_LIB"); lib != "" {
		return lib
	}
	candidate, err := filepath.Abs(filepath.Join("..", "..", "lib", "libonnxruntime.so"))
	if err == nil {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	tb.Skip("set ONNXRUNTIME_LIB or run `make ort/cpu` to run these tests")
	return ""
}

// fixture draws a bright blob centered on a plain background.
func fixture() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 320, 320))
	bg := color.NRGBA{G: 90, R: 40, A: 255}
	for y := 0; y < 320; y++ {
		for x := 0; x < 320; x++ {
			img.SetNRGBA(x, y, bg)
		}
	}
	blob := color.NRGBA{R: 230, G: 180, B: 150, A: 255}
	for y := 60; y < 260; y++ {
		for x := 110; x < 210; x++ {
			img.SetNRGBA(x, y, blob)
		}
	}
	return img
}
