package rmbg

import (
	"encoding/binary"
	"image"
)

// jpegOrientation returns the EXIF orientation (1..8) of a JPEG, or 1 if none
// is present or the EXIF data is malformed.
func jpegOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}
	i := 2
	for i+4 <= len(data) {
		if data[i] != 0xFF {
			i++
			continue
		}
		marker := data[i+1]
		// Standalone markers without a length payload.
		if marker == 0xD8 || marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7) {
			i += 2
			continue
		}
		if marker == 0xD9 || marker == 0xDA { // EOI or start of scan
			break
		}
		segLen := int(binary.BigEndian.Uint16(data[i+2 : i+4]))
		if segLen < 2 || i+2+segLen > len(data) {
			break
		}
		if marker == 0xE1 {
			if o := exifOrientation(data[i+4 : i+2+segLen]); o != 0 {
				return o
			}
		}
		i += 2 + segLen
	}
	return 1
}

// exifOrientation extracts the orientation tag from an APP1 EXIF payload.
func exifOrientation(b []byte) int {
	if len(b) < 6 || string(b[:6]) != "Exif\x00\x00" {
		return 0
	}
	tiff := b[6:]
	if len(tiff) < 8 {
		return 0
	}

	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 0
	}
	if order.Uint16(tiff[2:4]) != 0x002A {
		return 0
	}
	off := int(order.Uint32(tiff[4:8]))
	if off < 8 || off+2 > len(tiff) {
		return 0
	}

	count := int(order.Uint16(tiff[off : off+2]))
	entry := off + 2
	for n := 0; n < count; n++ {
		if entry+12 > len(tiff) {
			return 0
		}
		e := tiff[entry : entry+12]
		if order.Uint16(e[0:2]) == 0x0112 {
			v := int(order.Uint16(e[8:10]))
			if v >= 1 && v <= 8 {
				return v
			}
			return 0
		}
		entry += 12
	}
	return 0
}

// applyOrientation rotates/flips img according to the EXIF orientation value.
func applyOrientation(img image.Image, orientation int) image.Image {
	if orientation <= 1 || orientation > 8 {
		return img
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if orientation >= 5 { // orientations 5..8 swap the axes
		w, h = h, w
	}
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	src := toNRGBA(img)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var sx, sy int
			switch orientation {
			case 2: // mirrored horizontally
				sx, sy = b.Dx()-1-x, y
			case 3: // rotated 180
				sx, sy = b.Dx()-1-x, b.Dy()-1-y
			case 4: // mirrored vertically
				sx, sy = x, b.Dy()-1-y
			case 5: // mirrored then rotated 90 CW
				sx, sy = y, x
			case 6: // rotated 90 CW
				sx, sy = y, b.Dy()-1-x
			case 7: // mirrored then rotated 90 CCW
				sx, sy = b.Dx()-1-y, b.Dy()-1-x
			case 8: // rotated 90 CCW
				sx, sy = b.Dx()-1-y, x
			}
			dst.SetNRGBA(x, y, src.NRGBAAt(b.Min.X+sx, b.Min.Y+sy))
		}
	}
	return dst
}
