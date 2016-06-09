package exif

import (
	"bytes"
	"os"
	"testing"
)

var fns = []string{
	"coffee-sf.jpg",
	"gocon-tokyo.jpg",
	"sub.jpg",
}

func TestDecode(t *testing.T) {
	for _, n := range fns {
		testDecodeBytes(t, n)
	}
}

func testDecodeBytes(t *testing.T, fn string) {
	t.Log(fn)

	f, err := os.Open("../testdata/" + fn)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	raw, err := exifFromReader(f)
	if err != nil {
		t.Fatal(err)
	}

	x, err := DecodeBytes(raw)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(Sdump(x))
}

func TestEncodeBytes(t *testing.T) {
	for _, n := range fns {
		testEncodeBytes(t, n)
	}
}

func testEncodeBytes(t *testing.T, fn string) {
	t.Log(fn)

	f, err := os.Open("../testdata/" + fn)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	src, err := exifFromReader(f)
	if err != nil {
		t.Fatal(err)
	}

	x, err := DecodeBytes(src)
	if err != nil {
		t.Fatal(err)
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

func testDirEqual(t *testing.T, name string, a, b Dir) {
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
