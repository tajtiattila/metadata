package exifutil

import (
	"bytes"
	"fmt"
	"io"

	"github.com/tajtiattila/metadata/exif"
	"github.com/tajtiattila/metadata/exif/exiftag"
)

func Fdump(w io.Writer, x *exif.Exif) {
	showTags(w, "IFD0", exiftag.Tiff, x.IFD0)
	showTags(w, "IFD1", exiftag.Tiff, x.IFD1)
	showTags(w, "Exif", exiftag.Exif, x.Exif)
	showTags(w, "GPS", exiftag.GPS, x.GPS)
	showTags(w, "Interop", exiftag.Interop, x.Interop)

	if x.Thumb != nil {
		fmt.Fprintf(w, "thumb: %v bytes", len(x.Thumb))
	}
}

func Sdump(x *exif.Exif) string {
	buf := new(bytes.Buffer)
	Fdump(buf, x)
	return buf.String()
}

func showTags(w io.Writer, pfx string, dir uint32, d []exif.Entry) {
	if len(d) == 0 {
		return
	}
	fmt.Fprintln(w, pfx+":")
	for _, tag := range d {
		s := fmtName(dir, tag.Tag, 20)
		f, g := fmtTypeGrp(tag.Type, tag.Count)
		fmt.Fprintf(w, "  %s %s: %v\n", s, f, hexBytes(tag.Value, g))
	}
}

func hexBytes(p []byte, grp int) string {
	buf := new(bytes.Buffer)
	buf.WriteRune('[')
	for i, b := range p {
		if i != 0 && i%grp == 0 {
			buf.WriteRune(' ')
		}
		fmt.Fprintf(buf, "%02x", b)
	}
	buf.WriteRune(']')
	return buf.String()
}

func fmtName(dir uint32, tag uint16, maxlen int) string {
	id := exiftag.Id(dir | uint32(tag))
	return fmt.Sprintf("%04x %-*.*s", tag, maxlen, maxlen, id)
}

func fmtTypeGrp(typ uint16, count uint32) (string, int) {
	var n string
	var g int
	switch typ {
	case exif.TypeByte:
		n, g = "b", 1
	case exif.TypeAscii:
		n, g = "a", 1
	case exif.TypeShort:
		n, g = "s", 2
	case exif.TypeLong:
		n, g = "l", 4
	case exif.TypeRational:
		n, g = "r", 4
	case exif.TypeUndef:
		n, g = "u", 1
	case exif.TypeSLong:
		n, g = "L", 4
	case exif.TypeSRational:
		n, g = "R", 4
	case exif.TypeSByte:
		n, g = "B", 1
	case exif.TypeSShort:
		n, g = "S", 2
	case exif.TypeFloat:
		n, g = "f", 4
	case exif.TypeDouble:
		n, g = "f", 8
	default:
		n, g = "?", 1
	}
	return fmt.Sprintf("%d%s", count, n), g
}
