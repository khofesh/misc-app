package rmbg

import (
	"bytes"
	"fmt"
	"image"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register the webp decoder
)

// Decode reads a JPEG, PNG, or WebP image and applies the EXIF orientation for
// JPEGs.
func Decode(data []byte) (image.Image, error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decoding image: %w", err)
	}
	if format == "jpeg" {
		if o := jpegOrientation(data); o > 1 {
			img = applyOrientation(img, o)
		}
	}
	return img, nil
}

// Preprocess resizes img to size×size and returns an NCHW float32 tensor of
// shape [1,3,size,size] normalized with the given mean and std.
func Preprocess(img image.Image, size int, mean, std [3]float32) ([]float32, error) {
	if size <= 0 {
		return nil, fmt.Errorf("invalid model input size %d", size)
	}

	src := toNRGBA(img)
	dst := image.NewNRGBA(image.Rect(0, 0, size, size))
	draw.BiLinear.Scale(dst, dst.Rect, src, src.Bounds(), draw.Src, nil)

	out := make([]float32, 3*size*size)
	plane := size * size
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			i := y*dst.Stride + x*4
			off := y*size + x
			out[off] = (float32(dst.Pix[i+0])/255 - mean[0]) / std[0]
			out[plane+off] = (float32(dst.Pix[i+1])/255 - mean[1]) / std[1]
			out[2*plane+off] = (float32(dst.Pix[i+2])/255 - mean[2]) / std[2]
		}
	}
	return out, nil
}

// toNRGBA converts any image to a non-premultiplied RGBA image.
func toNRGBA(img image.Image) *image.NRGBA {
	if n, ok := img.(*image.NRGBA); ok {
		return n
	}
	b := img.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Rect, img, b.Min, draw.Src)
	return dst
}
