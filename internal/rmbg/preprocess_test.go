package rmbg

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestPreprocessLayoutAndNormalization(t *testing.T) {
	img := solid(4, 4, color.NRGBA{R: 255, A: 255})
	mean := [3]float32{0.5, 0.5, 0.5}
	std := [3]float32{0.5, 0.5, 0.5}

	out, err := Preprocess(img, 2, mean, std)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3*2*2 {
		t.Fatalf("len = %d, want 12", len(out))
	}

	plane := 4
	for i := 0; i < plane; i++ {
		if diff := out[i] - 1; diff > 1e-6 || diff < -1e-6 { // (1-0.5)/0.5
			t.Errorf("R plane[%d] = %v, want 1", i, out[i])
		}
		if diff := out[plane+i] + 1; diff > 1e-6 || diff < -1e-6 { // (0-0.5)/0.5
			t.Errorf("G plane[%d] = %v, want -1", i, out[plane+i])
		}
		if diff := out[2*plane+i] + 1; diff > 1e-6 || diff < -1e-6 {
			t.Errorf("B plane[%d] = %v, want -1", i, out[2*plane+i])
		}
	}
}

func TestPreprocessInvalidSize(t *testing.T) {
	if _, err := Preprocess(solid(1, 1, color.NRGBA{}), 0, imagenetMean, imagenetStd); err == nil {
		t.Fatal("expected error for size 0")
	}
}

func TestDecodePNG(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, solid(3, 2, color.NRGBA{G: 255, A: 255})); err != nil {
		t.Fatal(err)
	}
	img, err := Decode(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if got := img.Bounds(); got.Dx() != 3 || got.Dy() != 2 {
		t.Errorf("bounds = %v, want 3x2", got)
	}
}

func TestDecodeInvalid(t *testing.T) {
	if _, err := Decode([]byte("not an image")); err == nil {
		t.Fatal("expected an error")
	}
}

func TestApplyOrientation(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{B: 255, A: 255})

	// Orientation 6 rotates 90° clockwise: the top-left pixel moves to the
	// top-right, and the source left column becomes the output top row.
	got := applyOrientation(img, 6)
	if got.Bounds().Dx() != 1 || got.Bounds().Dy() != 2 {
		t.Fatalf("bounds = %v, want 1x2", got.Bounds())
	}
	if c := got.(*image.NRGBA).NRGBAAt(0, 0); c.R != 255 {
		t.Errorf("top pixel = %v, want red", c)
	}
	if c := got.(*image.NRGBA).NRGBAAt(0, 1); c.B != 255 {
		t.Errorf("bottom pixel = %v, want blue", c)
	}
}
