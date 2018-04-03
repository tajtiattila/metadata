package jpeg

import (
	"bytes"
	"io"

	"github.com/tajtiattila/metadata/metaio"
)

func init() {
	metaio.RegisterContainerFormat("jpeg", "\xff\xd8\xff", jpegFmt{})
}

var jpegExifPfx = []byte("Exif\x00\x00")
var jpegXMPPfx = []byte("http://ns.adobe.com/xap/1.0/\x00")

var jfifChunkHeader = []byte("JFIF\x00")
var jfxxChunkHeader = []byte("JFXX\x00")

type jpegFmt struct{}

var _ metaio.ContainerFormat = jpegFmt{}

func (jpegFmt) Scan(r io.Reader, f func(name string, data []byte)) (map[string]interface{}, error) {
	j, err := NewScanner(r)
	if err != nil {
		return nil, err
	}

	for j.NextChunk() {
		p := j.Bytes()
		if len(p) < 4 || p[0] != 0xff || p[1] != 0xe1 {
			continue
		}

		var name string
		var trim int
		switch {
		case j.IsChunk(0xe1, jpegExifPfx):
			name, trim = "exif", len(jpegExifPfx)
		case j.IsChunk(0xe1, jpegXMPPfx):
			name, trim = "xmp", len(jpegXMPPfx)
		}

		if name == "" {
			continue
		}

		_, p, err := j.ReadChunk()
		if err != nil {
			return nil, err
		}

		f(name, p[trim:])
	}

	return nil, nil
}

func (jpegFmt) WriteWithMeta(w io.Writer, r io.Reader, m []metaio.EncodedMeta) error {
	j, err := NewScanner(r)
	if err != nil {
		return err
	}

	var metaChunks [][]byte

	for _, rm := range m {
		switch rm.Name {
		case "exif":
			p := make([]byte, len(jpegExifPfx)+len(rm.Bytes))
			n := copy(p, jpegExifPfx)
			copy(p[n:], rm.Bytes)
			metaChunks = append(metaChunks, p)
		case "xmp":
			p := make([]byte, len(jpegXMPPfx)+len(rm.Bytes))
			n := copy(p, jpegXMPPfx)
			copy(p[n:], rm.Bytes)
			metaChunks = append(metaChunks, p)
		}
	}

	var segments [][]byte
	var jfifSeg, jfxxSeg []byte

	const (
		hasJFIF = uint(1 << iota)
		hasJFXX
		hasExif
		hasXMP
	)

	for j.Next() {
		seg, err := j.ReadSegment()
		if err != nil {
			return err
		}

		switch {

		case jfifSeg == nil && isChunkSegment(seg, 0xe0, jfifChunkHeader):
			jfifSeg = seg

		case jfxxSeg == nil && isChunkSegment(seg, 0xe0, jfxxChunkHeader):
			jfxxSeg = seg

		case isChunkSegment(seg, 0xe1, jpegExifPfx),
			isChunkSegment(seg, 0xe1, jpegXMPPfx):
			// pass

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

	for _, c := range metaChunks {
		err := WriteChunk(w, 0xe1, c)
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
