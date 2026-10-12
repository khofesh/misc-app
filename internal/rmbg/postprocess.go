package rmbg

import (
	"fmt"
	"image"
	"image/color"
	"strconv"
	"strings"
)

// NormalizeMask min-max normalizes the raw model output to [0,1]. If all
// values are equal the result is all ones (the whole image is foreground),
// matching the reference implementation's behavior on a zero-width range.
func NormalizeMask(data []float32) []float32 {
	out := make([]float32, len(data))
	if len(data) == 0 {
		return out
	}
	lo, hi := data[0], data[0]
	for _, v := range data {
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
	}
	if hi == lo {
		for i := range out {
			out[i] = 1
		}
		return out
	}
	scale := 1 / (hi - lo)
	for i, v := range data {
		out[i] = (v - lo) * scale
	}
	return out
}

// ResizeMask resizes a single-channel float mask with bilinear interpolation.
func ResizeMask(src []float32, sw, sh, dw, dh int) []float32 {
	if sw <= 0 || sh <= 0 || dw <= 0 || dh <= 0 {
		return nil
	}
	dst := make([]float32, dw*dh)
	xScale := float32(sw) / float32(dw)
	yScale := float32(sh) / float32(dh)

	for y := 0; y < dh; y++ {
		sy := (float32(y)+0.5)*yScale - 0.5
		y0 := int(sy)
		if sy < 0 {
			y0 = 0
			sy = 0
		}
		y1 := y0 + 1
		if y1 >= sh {
			y1 = sh - 1
		}
		fy := sy - float32(y0)
		if y0 >= sh {
			y0 = sh - 1
		}

		for x := 0; x < dw; x++ {
			sx := (float32(x)+0.5)*xScale - 0.5
			x0 := int(sx)
			if sx < 0 {
				x0 = 0
				sx = 0
			}
			x1 := x0 + 1
			if x1 >= sw {
				x1 = sw - 1
			}
			fx := sx - float32(x0)
			if x0 >= sw {
				x0 = sw - 1
			}

			v00 := src[y0*sw+x0]
			v10 := src[y0*sw+x1]
			v01 := src[y1*sw+x0]
			v11 := src[y1*sw+x1]
			top := v00 + (v10-v00)*fx
			bot := v01 + (v11-v01)*fx
			dst[y*dw+x] = top + (bot-top)*fy
		}
	}
	return dst
}

// MaskToGray converts a normalized mask to an 8-bit grayscale image.
func MaskToGray(mask []float32, w, h int) *image.Gray {
	g := image.NewGray(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := mask[y*w+x]
			if v < 0 {
				v = 0
			} else if v > 1 {
				v = 1
			}
			g.Pix[y*g.Stride+x] = uint8(v*255 + 0.5)
		}
	}
	return g
}

// Threshold turns a soft mask into an alpha of either 0 or 255. Levels outside
// 1..254 are ignored.
func Threshold(g *image.Gray, level int) {
	if level <= 0 || level >= 255 {
		return
	}
	for i, v := range g.Pix {
		if int(v) >= level {
			g.Pix[i] = 255
		} else {
			g.Pix[i] = 0
		}
	}
}

// ApplyAlpha returns a copy of img whose alpha is masked. Existing alpha is
// multiplied with the mask.
func ApplyAlpha(img image.Image, mask *image.Gray) *image.NRGBA {
	src := toNRGBA(img)
	b := src.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	mb := mask.Bounds()
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			c := src.NRGBAAt(b.Min.X+x, b.Min.Y+y)
			a := uint16(c.A)
			if x < mb.Dx() && y < mb.Dy() {
				a = a * uint16(mask.GrayAt(mb.Min.X+x, mb.Min.Y+y).Y) / 255
			}
			out.SetNRGBA(x, y, color.NRGBA{R: c.R, G: c.G, B: c.B, A: uint8(a)})
		}
	}
	return out
}

// Blend composites img over the solid color bg using mask as alpha. The result
// is opaque.
func Blend(img image.Image, mask *image.Gray, bg color.NRGBA) *image.NRGBA {
	src := toNRGBA(img)
	b := src.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	mb := mask.Bounds()
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			c := src.NRGBAAt(b.Min.X+x, b.Min.Y+y)
			a := float32(c.A) / 255
			if x < mb.Dx() && y < mb.Dy() {
				a *= float32(mask.GrayAt(mb.Min.X+x, mb.Min.Y+y).Y) / 255
			}
			out.SetNRGBA(x, y, color.NRGBA{
				R: blend(c.R, bg.R, a),
				G: blend(c.G, bg.G, a),
				B: blend(c.B, bg.B, a),
				A: 255,
			})
		}
	}
	return out
}

func blend(fg, bg uint8, a float32) uint8 {
	v := float32(fg)*a + float32(bg)*(1-a)
	if v < 0 {
		v = 0
	} else if v > 255 {
		v = 255
	}
	return uint8(v + 0.5)
}

// ParseHexColor parses "#rgb", "#rrggbb", or "#rrggbbaa" (leading '#' optional).
func ParseHexColor(s string) (color.NRGBA, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	expand := func(str string) (uint8, error) {
		v, err := strconv.ParseUint(str, 16, 8)
		return uint8(v), err
	}

	var r, g, b, a uint8 = 0, 0, 0, 255
	var err error
	switch len(s) {
	case 3:
		if r, err = expand(strings.Repeat(string(s[0]), 2)); err != nil {
			break
		}
		if g, err = expand(strings.Repeat(string(s[1]), 2)); err != nil {
			break
		}
		b, err = expand(strings.Repeat(string(s[2]), 2))
	case 6, 8:
		if r, err = expand(s[0:2]); err != nil {
			break
		}
		if g, err = expand(s[2:4]); err != nil {
			break
		}
		if b, err = expand(s[4:6]); err != nil {
			break
		}
		if len(s) == 8 {
			a, err = expand(s[6:8])
		}
	default:
		return color.NRGBA{}, fmt.Errorf("invalid hex color %q", s)
	}
	if err != nil {
		return color.NRGBA{}, fmt.Errorf("invalid hex color %q: %w", s, err)
	}
	return color.NRGBA{R: r, G: g, B: b, A: a}, nil
}
