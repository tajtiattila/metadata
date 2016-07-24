package exif

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/tajtiattila/metadata/exif/exiftag"
)

func Fdump(w io.Writer, x *Exif) {
	showTags(w, "IFD0", exiftag.Tiff, x.IFD0)
	showTags(w, "IFD1", exiftag.Tiff, x.IFD1)
	showTags(w, "Exif", exiftag.Exif, x.Exif)
	showTags(w, "GPS", exiftag.GPS, x.GPS)
	showTags(w, "Interop", exiftag.Interop, x.Interop)

	if x.Thumb != nil {
		fmt.Fprintf(w, "thumb: %v bytes", len(x.Thumb))
	}
}

func Sdump(x *Exif) string {
	buf := new(bytes.Buffer)
	Fdump(buf, x)
	return buf.String()
}

func showTags(w io.Writer, pfx string, dir uint32, d Dir) {
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
	case TypeByte:
		n, g = "b", 1
	case TypeAscii:
		n, g = "a", 1
	case TypeShort:
		n, g = "s", 2
	case TypeLong:
		n, g = "l", 4
	case TypeRational:
		n, g = "r", 4
	case TypeUndef:
		n, g = "u", 1
	case TypeSLong:
		n, g = "L", 4
	case TypeSRational:
		n, g = "R", 4
	case TypeSByte:
		n, g = "B", 1
	case TypeSShort:
		n, g = "S", 2
	case TypeFloat:
		n, g = "f", 4
	case TypeDouble:
		n, g = "f", 8
	default:
		n, g = "?", 1
	}
	return fmt.Sprintf("%d%s", count, n), g
}

type Formatter struct {
	binary.ByteOrder
}

func (f *Formatter) RawValue(typ uint16, cnt uint32, p []byte) string {
	var g int
	switch typ {

	default:
		fallthrough

	case TypeAscii, TypeByte, TypeUndef, TypeSByte:
		g = 1

	case TypeShort, TypeSShort:
		g = 2

	case TypeLong, TypeSLong, TypeRational, TypeSRational, TypeFloat, TypeDouble:
		g = 4
	}

	l := typeSize(typ, cnt)
	buf := new(bytes.Buffer)
	buf.WriteRune('[')
	for i := 0; i < l; i++ {
		if i != 0 && i%g == 0 {
			buf.WriteRune(' ')
		}
		if i < len(p) {
			fmt.Fprintf(buf, "%02x", p[i])
		} else {
			buf.WriteString("--")
		}
	}
	buf.WriteRune(']')
	return buf.String()
}

func (f *Formatter) Value(typ uint16, count uint32, p []byte) string {
	n := typeSize(typ, count)
	if n < 0 || len(p) < n {
		// show raw value for invalid Entry
		return f.RawValue(typ, count, p)
	}

	var values []interface{}
	var x interface{}
	cnt := int(count)

	switch typ {

	default:
		fallthrough

	case TypeByte, TypeUndef, TypeSByte:
		return fmt.Sprintf("% 2x", p)

	case TypeAscii:
		l := int(cnt)
		if l < 1 || p[l-1] != 0 {
			// Ascii too short or without NUL
			return f.RawValue(typ, count, p)
		}
		return fmt.Sprintf("%q", p[:l-1])

	case TypeRational, TypeSRational:
		for i := 0; i < cnt; i++ {
			num := f.ByteOrder.Uint32(p[8*i:])
			den := f.ByteOrder.Uint32(p[8*i+4:])
			if typ == TypeRational {
				x = fmt.Sprintf("%d/%d", num, den)
			} else {
				x = fmt.Sprintf("%d/%d", int32(num), int32(den))
			}
			values = append(values, x)
		}

	case TypeShort, TypeSShort:
		for i := 0; i < cnt; i++ {
			v := f.ByteOrder.Uint16(p[2*i:])
			if typ == TypeSShort {
				x = int16(v)
			} else {
				x = v
			}
			values = append(values, x)
		}

	case TypeLong, TypeSLong, TypeFloat:
		for i := 0; i < cnt; i++ {
			v := f.ByteOrder.Uint32(p[4*i:])
			switch typ {
			case TypeSLong:
				x = int16(v)
			case TypeFloat:
				x = math.Float32frombits(v)
			default: // TypeLong
				x = v
			}
			values = append(values, x)
		}

	case TypeDouble:
		for i := 0; i < cnt; i++ {
			values = append(values, f.ByteOrder.Uint64(p[8*i:]))
		}
	}

	buf := new(bytes.Buffer)
	buf.WriteRune('[')
	for i, e := range values {
		if i != 0 {
			buf.WriteRune(' ')
		}
		fmt.Fprint(buf, e)
	}
	buf.WriteRune(']')
	return buf.String()
}
