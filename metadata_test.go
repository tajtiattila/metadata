package metadata_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/tajtiattila/metadata"
	"github.com/tajtiattila/metadata/exif"
	xjpeg "github.com/tajtiattila/metadata/jpeg"
	"github.com/tajtiattila/metadata/testutil"
)

func TestParse(t *testing.T) {
	fileList := testutil.MediaFileInfos(t)

	for _, e := range fileList {
		testParse(t, e)
	}
}

func testParse(t *testing.T, e testutil.FileInfo) {
	if _, ok := e["Error"]; ok {
		// exiftool found error parse either
		return
	}

	fn, ok := e.String("SourceFile")
	if !ok {
		return
	}

	if _, ok := e.String("CreateDate"); !ok {
		// exiftool found no metadata either
		return
	}

	f, err := os.Open(fn)
	if err != nil {
		t.Errorf("Open %s: error %v", fn, err)
		return
	}
	defer f.Close()

	m, err := metadata.Parse(f)
	if err != nil {
		if err == metadata.ErrUnknownFormat {
			// format not (yet?) supported
			return
		}
		t.Errorf("metadata.Parse %s: error %v", fn, err)
		f.Seek(0, 0)
		dumpJpeg(t, f)
		return
	}
	m.Get(metadata.DateTimeCreated)
}

var jpegExifPfx = []byte("Exif\x00\x00")
var jpegXMPPfx = []byte("http://ns.adobe.com/xap/1.0/\x00")

func dumpJpeg(t *testing.T, r io.Reader) {
	j, err := xjpeg.NewScanner(r)
	if err != nil {
		t.Error("jpeg.NewScanner:", err)
		return
	}

	exiff := func(p []byte) {
		dumpExifBytes(t, p)
	}
	xmpf := func(p []byte) {
		dumpXmpBytes(t, p)
	}

	for j.Next() {
		p := j.Bytes()
		if !j.StartChunk() {
			t.Logf("jpeg nochunk %.32x %.32q", p, p)
			continue
		}

		if len(p) < 4 || p[0] != 0xff || p[1] != 0xe1 {
			continue
		}

		var trim int
		var f func(p []byte)
		kind := "unknown"
		switch {
		case bytes.HasPrefix(p[4:], jpegExifPfx):
			trim, f, kind = len(jpegExifPfx), exiff, "exif"
		case bytes.HasPrefix(p[4:], jpegXMPPfx):
			trim, f, kind = len(jpegXMPPfx), xmpf, "xmp"
		}

		p, err := j.ReadChunk()
		if err != nil {
			t.Error("jpeg.Scanner.ReadChunk:", err)
			return
		}

		t.Logf("jpeg chunk %s %.32x %.32q", kind, p, p)
		if f == nil {
			continue
		}

		if f != nil {
			f(p[trim+4:])
		}
	}
}

func dumpExifBytes(t *testing.T, p []byte) {
	_, err := exif.DecodeBytes(p)
	if err != nil {
		t.Error("exif.Decode:", err)
		return
	}
}

func dumpXmpBytes(t *testing.T, p []byte) {
}
