// Package orient provides the Orient function
// that applies an Exif orientation to an image.
package orient

import (
	"image"
	"image/draw"
)

// Orient changes the orientation of im based on
// the Exif orientation value.
//
// It performes the following operation based on o:
//   2: flip horizontal
//   3: rotate 180°
//   4: flip vertical
//   5: transpose
//   6: rotate 90°
//   7: transverse (transpose and rotate 180°)
//   8: rotate 270°
//
// It will return either a new image for the values
// of o above, or im istelf otherwise.
func Orient(im image.Image, o int) image.Image {
	if o < 2 || o > 8 {
		return im
	}

	var dst *image.RGBA
	if o >= 5 {
		dst = transpose(im)
		o -= 4
	} else {
		dst = asRGBA(im)
	}

	switch o {
	case 2:
		flipHorz(dst)
		return dst
	case 3:
		flipHorz(dst)
		flipVert(dst)
		return dst
	case 4:
		flipVert(dst)
		return dst
	}

	return dst
}

func asRGBA(src image.Image) *image.RGBA {
	db := src.Bounds().Canon()
	db = db.Sub(db.Min)
	dst := image.NewRGBA(db)
	draw.Draw(dst, db, src, src.Bounds().Min, draw.Src)
	return dst
}

func transpose(src image.Image) *image.RGBA {
	sz := src.Bounds().Size()
	o := src.Bounds().Canon().Min
	dst := image.NewRGBA(image.Rect(0, 0, sz.Y, sz.X))
	for y := 0; y < sz.Y; y++ {
		for x := 0; x < sz.X; x++ {
			c := src.At(o.X+x, o.Y+y)
			dst.Set(y, x, c)
		}
	}
	return dst
}

func flipHorz(im *image.RGBA) {
	w := im.Rect.Dx()
	nswap := w / 2
	i0, i1 := 0, (w-1)*4
	for y := im.Rect.Min.Y; y < im.Rect.Max.Y; y++ {
		x0, x1 := i0, i1
		for i := 0; i < nswap; i++ {
			for j := 0; j < 4; j++ {
				im.Pix[x0+j], im.Pix[x1+j] = im.Pix[x1+j], im.Pix[x0+j]
			}
			x0 += 4
			x1 -= 4
		}
		i0 += im.Stride
		i1 += im.Stride
	}
}

func flipVert(im *image.RGBA) {
	w := 4 * im.Rect.Dx()
	tmp := make([]uint8, w)
	ny := im.Rect.Dy()
	nswap := ny / 2
	for i := 0; i < nswap; i++ {
		o0 := i * im.Stride
		o1 := (ny - 1 - i) * im.Stride
		copy(tmp, im.Pix[o0:o0+w])
		copy(im.Pix[o0:o0+w], im.Pix[o1:o1+w])
		copy(im.Pix[o1:o1+w], tmp)
	}
}
