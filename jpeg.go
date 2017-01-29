package metadata

import (
	"io"

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
	for (ex == nil || xmp == nil) && j.NextChunk() {
		p := j.Bytes()
		if len(p) < 4 || p[0] != 0xff || p[1] != 0xe1 {
			continue
		}

		var pdata *[]byte
		var trim int
		switch {
		case ex == nil && j.IsChunk(0xe1, jpegExifPfx):
			pdata, trim = &ex, len(jpegExifPfx)
		case xmp == nil && j.IsChunk(0xe1, jpegXMPPfx):
			pdata, trim = &xmp, len(jpegXMPPfx)
		}

		if pdata == nil {
			continue
		}

		_, p, err := j.ReadChunk()
		if err != nil {
			return nil, err
		}

		*pdata = p[trim:]
	}

	if ex == nil && xmp == nil {
		if err = j.Err(); err != nil {
			return nil, err
		}
		return nil, ErrNoMeta
	}

	var meta []*Metadata
	var firstErr error

	if ex != nil {
		m, err := FromExifBytes(ex)
		if m != nil {
			meta = append(meta, m)
		}
		if err != nil {
			firstErr = err
		}
	}

	if xmp != nil {
		m, err := FromXMPBytes(xmp)
		if m != nil {
			meta = append(meta, m)
		}
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	if len(meta) == 0 {
		err := firstErr
		if err == nil {
			err = ErrNoMeta
		}
		return nil, err
	}

	return Merge(meta...), nil
}
