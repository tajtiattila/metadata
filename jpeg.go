package metadata

import (
	"bytes"
	"io"

	"github.com/tajtiattila/metadata/exif"
	xjpeg "github.com/tajtiattila/metadata/jpeg"
)

var jpegExifPfx = []byte("Exif\x00\x00")
var jpegXMPPfx = []byte("http://ns.adobe.com/xap/1.0/\x00")

func parseJpeg(r io.Reader) (*Metadata, error) {
	j, err := xjpeg.NewScanner(r)
	if err != nil {
		return nil, err
	}

	var ex, xmp []byte
	for (ex == nil || xmp == nil) && j.Next() {
		if !j.StartChunk() {
			continue
		}

		p := j.Bytes()
		if len(p) < 4 || p[0] != 0xff || p[1] != 0xe1 {
			continue
		}

		var pdata *[]byte
		var trim int
		switch {
		case ex == nil && bytes.HasPrefix(p[4:], jpegExifPfx):
			pdata, trim = &ex, len(jpegExifPfx)
		case xmp == nil && bytes.HasPrefix(p[4:], jpegXMPPfx):
			pdata, trim = &xmp, len(jpegXMPPfx)
		}

		if pdata == nil {
			continue
		}

		p, err := j.ReadChunk()
		if err != nil {
			return nil, err
		}

		*pdata = p[trim+4:]
	}

	err = j.Err()
	if err != nil {
		return nil, err
	}

	m := new(Metadata)

	if ex != nil {
		x, err := exif.DecodeBytes(ex)
		if err == nil {
			addExif(m, x)
		}
	}

	if xmp != nil {
		addXmp(m, xmp)
	}

	if len(m.Attr) == 0 {
		return nil, ErrNoMeta
	}

	return m, nil
}
