package orient

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"
	"testing"
)

func TestOrient(t *testing.T) {
	type format struct {
		name     string
		newImage func(r image.Rectangle) draw.Image
	}
	var formats = []format{
		{
			name: "RGBA",
			newImage: func(r image.Rectangle) draw.Image {
				return image.NewRGBA(r)
			},
		},
		{
			name: "Gray",
			newImage: func(r image.Rectangle) draw.Image {
				return image.NewGray(r)
			},
		},
		{
			name: "CMYK",
			newImage: func(r image.Rectangle) draw.Image {
				return image.NewCMYK(r)
			},
		},
	}

	for _, f := range formats {
		want := getPix(t, 1, f.newImage)
		for o := 1; o <= 8; o++ {
			src := getPix(t, o, f.newImage)
			got := Orient(src, o)
			if err := sameImage(got, want); err != nil {
				t.Errorf("%s orientation %v: %v", f.name, o, err)
			}

			wantt := src.Bounds().Dx() == want.Bounds().Dy()
			gott := IsTranspose(o)
			if wantt != gott {
				t.Errorf("IsTranspose(%v) reports %v, want %v", o, gott, wantt)
			}
		}
	}
}

var pixmap = map[int][][]bool{
	1: mkpix(`
. . 8 8 8 8 8 8 . .
. . 8 . . . . . . .
. . 8 . . . . . . .
. . 8 8 8 8 . . . .
. . 8 . . . . . . .
. . 8 . . . . . . .
. . 8 . . . . . . .
. . 8 . . . . . . .
`),
	2: mkpix(`
. . 8 8 8 8 8 8 . .
. . . . . . . 8 . .
. . . . . . . 8 . .
. . . . 8 8 8 8 . .
. . . . . . . 8 . .
. . . . . . . 8 . .
. . . . . . . 8 . .
. . . . . . . 8 . .
`),
	3: mkpix(`
. . . . . . . 8 . .
. . . . . . . 8 . .
. . . . . . . 8 . .
. . . . . . . 8 . .
. . . . 8 8 8 8 . .
. . . . . . . 8 . .
. . . . . . . 8 . .
. . 8 8 8 8 8 8 . .
`),
	4: mkpix(`
. . 8 . . . . . . .
. . 8 . . . . . . .
. . 8 . . . . . . .
. . 8 . . . . . . .
. . 8 8 8 8 . . . .
. . 8 . . . . . . .
. . 8 . . . . . . .
. . 8 8 8 8 8 8 . .
`),
	5: mkpix(`
. . . . . . . .
. . . . . . . .
8 8 8 8 8 8 8 8
8 . . 8 . . . .
8 . . 8 . . . .
8 . . 8 . . . .
8 . . . . . . .
8 . . . . . . .
. . . . . . . .
. . . . . . . .
`),
	6: mkpix(`
. . . . . . . .
. . . . . . . .
8 . . . . . . .
8 . . . . . . .
8 . . 8 . . . .
8 . . 8 . . . .
8 . . 8 . . . .
8 8 8 8 8 8 8 8
. . . . . . . .
. . . . . . . .
`),
	7: mkpix(`
. . . . . . . .
. . . . . . . .
. . . . . . . 8
. . . . . . . 8
. . . . 8 . . 8
. . . . 8 . . 8
. . . . 8 . . 8
8 8 8 8 8 8 8 8
. . . . . . . .
. . . . . . . .
`),
	8: mkpix(`
. . . . . . . .
. . . . . . . .
8 8 8 8 8 8 8 8
. . . . 8 . . 8
. . . . 8 . . 8
. . . . 8 . . 8
. . . . . . . 8
. . . . . . . 8
. . . . . . . .
. . . . . . . .
`),
}

func mkpix(s string) [][]bool {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	w := (len(lines[0]) + 1) / 2
	r := make([][]bool, len(lines))
	for y, line := range lines {
		row := make([]bool, w)
		for x := 0; x < w; x++ {
			pixel := line[x*2]
			if pixel != '.' && pixel != '8' {
				panic(fmt.Sprint(s, x, y, "invalid pix"))
			}
			row[x] = pixel == '8'
		}
		r[y] = row
	}
	return r
}

func getPix(t *testing.T, o int, f func(image.Rectangle) draw.Image) image.Image {
	r := pixmap[o]
	if r == nil {
		t.Fatal("invalid orientation", o)
	}

	const scale = 10
	rx, ry := len(r[0]), len(r)
	dx, dy := rx*scale, ry*scale
	im := f(image.Rect(0, 0, dx, dy))

	dot := color.RGBA{255, 255, 255, 255} // white
	eight := color.RGBA{255, 0, 0, 255}   // red
	for y := 0; y < dy; y++ {
		for x := 0; x < dx; x++ {
			var col color.Color
			if r[y/scale][x/scale] {
				col = eight
			} else {
				col = dot
			}
			im.Set(x, y, col)
		}
	}
	return im
}

func sameImage(got, want image.Image) error {
	if got.Bounds() != want.Bounds() {
		return fmt.Errorf("bounds differ: got %v want %v", got.Bounds(), want.Bounds())
	}
	r := got.Bounds()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			gp, wp := got.At(x, y), want.At(x, y)
			if !sameColor(gp, wp) {
				return fmt.Errorf("pixel at %d,%d differ: got %v want %v", x, y, gp, wp)
			}
		}
	}
	return nil
}

func sameColor(a, b color.Color) bool {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}
