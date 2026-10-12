package rmbg

import (
	"image"
	"image/color"
	"testing"
)

func TestNormalizeMask(t *testing.T) {
	tests := []struct {
		name string
		in   []float32
		want []float32
	}{
		{"range", []float32{-1, 0, 1}, []float32{0, 0.5, 1}},
		{"equal", []float32{3, 3, 3}, []float32{1, 1, 1}},
		{"empty", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeMask(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if diff := got[i] - tt.want[i]; diff > 1e-6 || diff < -1e-6 {
					t.Errorf("got[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestResizeMaskAverages(t *testing.T) {
	// 2x2 down to 1x1 is the average of the four values.
	src := []float32{0, 1, 0, 1}
	got := ResizeMask(src, 2, 2, 1, 1)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if diff := got[0] - 0.5; diff > 1e-6 || diff < -1e-6 {
		t.Errorf("got %v, want 0.5", got[0])
	}
}

func TestResizeMaskIdentity(t *testing.T) {
	src := []float32{0.1, 0.2, 0.3, 0.4}
	got := ResizeMask(src, 2, 2, 2, 2)
	for i := range src {
		if diff := got[i] - src[i]; diff > 1e-6 || diff < -1e-6 {
			t.Errorf("got[%d] = %v, want %v", i, got[i], src[i])
		}
	}
}

func TestMaskToGray(t *testing.T) {
	g := MaskToGray([]float32{-1, 0.5, 2}, 3, 1)
	want := []uint8{0, 128, 255}
	for i, w := range want {
		if g.Pix[i] != w {
			t.Errorf("pix[%d] = %d, want %d", i, g.Pix[i], w)
		}
	}
}

func TestThreshold(t *testing.T) {
	g := image.NewGray(image.Rect(0, 0, 4, 1))
	copy(g.Pix, []byte{0, 100, 128, 255})

	Threshold(g, 128)
	want := []byte{0, 0, 255, 255}
	for i, w := range want {
		if g.Pix[i] != w {
			t.Errorf("pix[%d] = %d, want %d", i, g.Pix[i], w)
		}
	}
}

func TestThresholdNoOp(t *testing.T) {
	for _, level := range []int{0, 255} {
		g := image.NewGray(image.Rect(0, 0, 2, 1))
		copy(g.Pix, []byte{10, 200})
		Threshold(g, level)
		if g.Pix[0] != 10 || g.Pix[1] != 200 {
			t.Errorf("level %d changed the mask", level)
		}
	}
}

func solid(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func TestApplyAlpha(t *testing.T) {
	img := solid(2, 1, color.NRGBA{R: 255, A: 255})
	mask := image.NewGray(image.Rect(0, 0, 2, 1))
	mask.Pix[0] = 0
	mask.Pix[1] = 255

	out := ApplyAlpha(img, mask)
	if out.NRGBAAt(0, 0).A != 0 {
		t.Errorf("alpha at 0 = %d, want 0", out.NRGBAAt(0, 0).A)
	}
	if out.NRGBAAt(1, 0).A != 255 {
		t.Errorf("alpha at 1 = %d, want 255", out.NRGBAAt(1, 0).A)
	}
}

func TestApplyAlphaMultipliesExistingAlpha(t *testing.T) {
	img := solid(1, 1, color.NRGBA{R: 255, A: 200})
	mask := image.NewGray(image.Rect(0, 0, 1, 1))
	mask.Pix[0] = 128

	out := ApplyAlpha(img, mask)
	want := uint8(200 * 128 / 255)
	if out.NRGBAAt(0, 0).A != want {
		t.Errorf("alpha = %d, want %d", out.NRGBAAt(0, 0).A, want)
	}
}

func TestBlend(t *testing.T) {
	img := solid(3, 1, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	mask := image.NewGray(image.Rect(0, 0, 3, 1))
	mask.Pix[0] = 0
	mask.Pix[1] = 255
	mask.Pix[2] = 128

	out := Blend(img, mask, color.NRGBA{B: 0, A: 255})

	if got := out.NRGBAAt(0, 0); got.R != 0 || got.A != 255 {
		t.Errorf("fully masked = %v, want opaque black", got)
	}
	if got := out.NRGBAAt(1, 0); got.R != 255 || got.A != 255 {
		t.Errorf("fully foreground = %v, want opaque white", got)
	}
	if got := out.NRGBAAt(2, 0); got.R < 126 || got.R > 130 {
		t.Errorf("half masked R = %d, want ~128", got.R)
	}
}

func TestParseHexColor(t *testing.T) {
	tests := []struct {
		in      string
		want    color.NRGBA
		wantErr bool
	}{
		{"#ffffff", color.NRGBA{R: 255, G: 255, B: 255, A: 255}, false},
		{"ffffff", color.NRGBA{R: 255, G: 255, B: 255, A: 255}, false},
		{"#f00", color.NRGBA{R: 255, A: 255}, false},
		{"#11223344", color.NRGBA{R: 0x11, G: 0x22, B: 0x33, A: 0x44}, false},
		{"", color.NRGBA{}, true},
		{"#12345", color.NRGBA{}, true},
		{"#gggggg", color.NRGBA{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseHexColor(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tt.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
