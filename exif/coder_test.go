package exif

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/tajtiattila/metadata/exif/exiftag"
	"github.com/tajtiattila/metadata/testutil"
)

func TestDecode(t *testing.T) {
	for _, n := range testutil.MediaFileNames(t, "image/jpeg") {
		testDecodeBytes(t, n)
	}
}

func testDecodeBytes(t *testing.T, fn string) {
	t.Log(fn)

	f, err := os.Open(fn)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	raw, err := exifFromReader(f)
	if err != nil {
		if err == NotFound {
			return
		}
		t.Fatal(err)
	}

	x, err := DecodeBytes(raw)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(sdump(x))
}

func TestEncodeBytes(t *testing.T) {
	for _, n := range testutil.MediaFileNames(t, "image/jpeg") {
		testEncodeBytes(t, n)
	}
}

func testEncodeBytes(t *testing.T, fn string) {
	t.Log(fn)

	f, err := os.Open(fn)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	src, err := exifFromReader(f)
	if err != nil {
		if err == NotFound {
			return
		}
		t.Fatal("exifFromReader:", err)
	}

	x, err := DecodeBytes(src)
	if err != nil {
		t.Logf("%.32x", src)
		t.Fatal("DecodeBytes:", err)
	}

	enc, err := x.EncodeBytes()
	if err != nil {
		t.Fatal(err)
	}

	x2, err := DecodeBytes(enc)
	if err != nil {
		t.Fatal(err)
	}

	testExifEqual(t, x, x2)
}

func testExifEqual(t *testing.T, a, b *Exif) {
	testDirEqual(t, "IFD0", a.IFD0, b.IFD0)
	testDirEqual(t, "Exif", a.Exif, b.Exif)
	testDirEqual(t, "GPS", a.GPS, b.GPS)
	testDirEqual(t, "Interop", a.Interop, b.Interop)

	// check thumb IFD only if there is a thumb
	if len(a.Thumb) != 0 && len(b.Thumb) != 0 {
		testDirEqual(t, "IFD1", a.IFD1, b.IFD1)
	}
}

func testDirEqual(t *testing.T, name string, a, b []Entry) {
	if len(a) != len(b) {
		t.Errorf("%s length differ: %d != %d\n", name, len(a), len(b))
		return
	}
	for i := range a {
		ta, tb := a[i], b[i]
		if ta.Tag != tb.Tag || ta.Type != tb.Type || ta.Count != tb.Count {
			t.Errorf("%s tag %d differ: %+v != %+v\n", name, i, ta, tb)
			continue
		}
		switch ta.Tag {
		case ifd0exifSub,
			ifd0gpsSub,
			ifd0interopSub:
			continue
		}
		if !bytes.Equal(ta.Value, tb.Value) {
			t.Errorf("%s value %d differ: %v != %v\n", name, i, ta.Value, tb.Value)
		}
	}
}

func fdump(w io.Writer, x *Exif) {
	showTags(w, "IFD0", exiftag.Tiff, x.IFD0)
	showTags(w, "IFD1", exiftag.Tiff, x.IFD1)
	showTags(w, "Exif", exiftag.Exif, x.Exif)
	showTags(w, "GPS", exiftag.GPS, x.GPS)
	showTags(w, "Interop", exiftag.Interop, x.Interop)

	if x.Thumb != nil {
		fmt.Fprintf(w, "thumb: %v bytes", len(x.Thumb))
	}
}

func sdump(x *Exif) string {
	buf := new(bytes.Buffer)
	fdump(buf, x)
	return buf.String()
}

func showTags(w io.Writer, pfx string, dir uint32, d []Entry) {
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
