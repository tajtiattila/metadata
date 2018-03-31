package jpeg

import (
	"bytes"
	"io"

	"github.com/pkg/errors"
	"github.com/tajtiattila/metadata/driver"
)

func init() {
	driver.RegisterContainerFormat("jpeg", "\xff\xd8\xff", func() driver.Container {
		return new(container)
	})
}

var _ driver.Container = new(container)

type container struct {
	r io.Reader

	rawMeta []driver.RawMeta
}

var jpegExifPfx = []byte("Exif\x00\x00")
var jpegXMPPfx = []byte("http://ns.adobe.com/xap/1.0/\x00")

var jfifChunkHeader = []byte("JFIF\x00")
var jfxxChunkHeader = []byte("JFXX\x00")

func (c *container) Parse(r io.Reader) error {
	j, err := NewScanner(r)
	if err != nil {
		return err
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
			return err
		}

		*pdata = p[trim:]
	}

	c.r = r

	if ex != nil {
		c.rawMeta = append(c.rawMeta, driver.RawMeta{
			Name:  "exif",
			Bytes: ex,
		})
	}

	if xmp != nil {
		c.rawMeta = append(c.rawMeta, driver.RawMeta{
			Name:  "xmp",
			Bytes: xmp,
		})
	}

	return nil
}

func (c *container) WriteTo(w io.Writer) error {
	rs, ok := c.r.(io.ReadSeeker)
	if !ok {
		return driver.ErrNotReadSeeker
	}

	_, err := rs.Seek(0, io.SeekStart)
	if err != nil {
		return errors.Wrap(err, "jpeg: seek error")
	}

	j, err := NewScanner(rs)
	if err != nil {
		return err
	}

	var exifdata []byte
	var xmpdata []byte

	for _, rm := range c.rawMeta {
		switch rm.Name {
		case "exif":
			exifdata = make([]byte, len(jpegExifPfx)+len(rm.Bytes))
			n := copy(exifdata, jpegExifPfx)
			copy(exifdata[n:], rm.Bytes)
		case "xmp":
			xmpdata = make([]byte, len(jpegXMPPfx)+len(rm.Bytes))
			n := copy(xmpdata, jpegXMPPfx)
			copy(xmpdata[n:], rm.Bytes)
		}
	}

	var segments [][]byte
	var jfifSeg, jfxxSeg []byte
	hasMask := uint(0)

	const (
		hasJFIF = uint(1 << iota)
		hasJFXX
		hasExif
		hasXMP
	)

	for hasMask != (hasJFIF|hasJFXX|hasExif|hasXMP) && j.Next() {
		seg, err := j.ReadSegment()
		if err != nil {
			return err
		}

		switch {

		case jfifSeg == nil && isChunkSegment(seg, 0xe0, jfifChunkHeader):
			hasMask |= hasJFIF
			jfifSeg = seg

		case jfxxSeg == nil && isChunkSegment(seg, 0xe0, jfxxChunkHeader):
			hasMask |= hasJFXX
			jfxxSeg = seg

		case isChunkSegment(seg, 0xe1, jpegExifPfx):
			hasMask |= hasExif
			if exifdata == nil {
				exifdata = seg[4:]
			}

		case isChunkSegment(seg, 0xe1, jpegXMPPfx):
			hasMask |= hasXMP
			if xmpdata == nil {
				xmpdata = seg[4:]
			}

		default:
			segments = append(segments, seg)
		}
	}
	if err := j.Err(); err != nil {
		return err
	}

	// write segments in standard jpeg/jfif header order
	ww := errw{w: w}
	ww.write(segments[0])
	ww.write(jfifSeg)
	ww.write(jfxxSeg)

	if exifdata != nil {
		err := WriteChunk(w, 0xe1, exifdata)
		if err != nil {
			return err
		}
	}

	if xmpdata != nil {
		err := WriteChunk(w, 0xe1, xmpdata)
		if err != nil {
			return err
		}
	}

	// write other segments in jpeg (DCT, COM, APP1/XMP...)
	for _, seg := range segments[1:] {
		ww.write(seg)
	}

	if ww.err != nil {
		return ww.err
	}

	// copy bytes unread so far, such as actual image data
	_, err = io.Copy(w, j.Reader())
	return err
}

func isChunkSegment(seg []byte, marker byte, pfx []byte) bool {
	if len(seg) >= 4 && seg[0] == '\xff' && seg[1] == marker {
		return bytes.HasPrefix(seg[4:], pfx)
	}
	return false
}

type errw struct {
	w   io.Writer
	err error
}

func (w *errw) write(p []byte) {
	_, w.err = w.w.Write(p)
}
