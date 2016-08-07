package exif_test

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"

	"github.com/tajtiattila/metadata/exif"
)

func TestNewImageLatLong(t *testing.T) {
	const (
		size   = 100
		border = 30
	)
	im := image.NewRGBA(image.Rect(0, 0, size, size))
	for x := 0; x < size; x++ {
		for y := 0; y < size; y++ {
			var c color.RGBA
			if x < border || size-border <= x ||
				y < border || size-border <= y {
				c = color.RGBA{255, 255, 255, 255}
			} else {
				c = color.RGBA{255, 0, 0, 255}
			}
			im.Set(x, y, c)
		}
	}

	jbuf := new(bytes.Buffer)
	if err := jpeg.Encode(jbuf, im, nil); err != nil {
		t.Fatal("image encode:", err)
	}

	x := exif.New(size, size)
	const (
		lat float64 = 51.5125
		lon         = -0.125
	)
	x.SetLatLong(lat, lon)

	xenc, err := x.EncodeBytes()
	if err != nil {
		t.Fatal("EncodeBytes:", err)
	}
	_, err = exif.DecodeBytes(xenc)
	if err != nil {
		t.Logf("%x", xenc)
		t.Fatal("DecodeBytes:", err)
	}

	xbuf := new(bytes.Buffer)
	if err := exif.Copy(xbuf, bytes.NewReader(jbuf.Bytes()), x); err != nil {
		t.Fatal("exif.Copy:", err)
	}

	if _, _, err := image.Decode(bytes.NewReader(xbuf.Bytes())); err != nil {
		t.Fatal("image decode:", err)
	}

	t.Logf("%x", xbuf.Bytes())

	x, err = exif.Decode(bytes.NewReader(xbuf.Bytes()))
	if err != nil {
		t.Fatal("exif.Decode:", err)
	}

	xlat, xlon, ok := x.LatLong()
	if !ok {
		t.Fatal("exif has no lat/long")
	}
	if xlat != lat {
		t.Errorf("exif has lat=%v, want %v", xlat, lat)
	}
	if xlon != lon {
		t.Errorf("exif has lon=%v, want %v", xlon, lon)
	}
}
