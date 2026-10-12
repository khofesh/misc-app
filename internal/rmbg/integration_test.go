//go:build integration && linux

package rmbg

import (
	"image"
	"testing"
)

func meanAlpha(img image.Image, x0, y0, x1, y1 int) float64 {
	var sum, n float64
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			sum += float64(a >> 8)
			n++
		}
	}
	return sum / n
}

func TestRemoveIntegration(t *testing.T) {
	r, err := New(Options{Model: "u2netp", Device: DeviceCPU, ORTLibPath: ortLibPath(t)})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	if r.Device() != DeviceCPU {
		t.Fatalf("device = %s, want cpu", r.Device())
	}

	img := fixture()
	out, err := r.Remove(img)
	if err != nil {
		t.Fatal(err)
	}
	if out.Bounds() != img.Bounds() {
		t.Fatalf("bounds = %v, want %v", out.Bounds(), img.Bounds())
	}

	b := out.Bounds()
	corner := meanAlpha(out, 0, 0, 40, 40)
	center := meanAlpha(out, b.Dx()/2-40, b.Dy()/2-40, b.Dx()/2+40, b.Dy()/2+40)
	if corner > 10 {
		t.Errorf("corner mean alpha = %.1f, want mostly transparent", corner)
	}
	if center < 200 {
		t.Errorf("center mean alpha = %.1f, want mostly opaque", center)
	}
}

func BenchmarkRemove(b *testing.B) {
	r, err := New(Options{Model: "u2netp", Device: DeviceCPU, ORTLibPath: ortLibPath(b)})
	if err != nil {
		b.Fatal(err)
	}
	defer r.Close()

	img := fixture()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := r.Remove(img); err != nil {
			b.Fatal(err)
		}
	}
}
