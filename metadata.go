package metadata

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

const (
	// Date/time values, with optional time zone
	// but otherwise formatted as RFC3339.
	//
	// Note: Exif has no standard way to specify time zone,
	// GPS location can be used to deduce it. From Exif the corresponding
	// SubSecTime is included in the values reported.
	DateTimeOriginal = "DateTimeOriginal" // date of original image (eg. scanned photo)
	DateTimeCreated  = "DateTimeCreated"  // original file creation date (eg. time of scan)

	GPSDateTime = "GPSDateTime" // Date/time of GPS fix (RFC3339, always UTC)

	// latitude and longitude are signed floating point
	// values formatted with no exponent
	GPSLatitude  = "GPSLatitude"  // +north, -south
	GPSLongitude = "GPSLongitude" // +east, -west

	// Orientation (integer) 1..8, values are like exif
	Orientation = "Orientation"

	// XMP Rating (integer), -1: rejected, 0: unrated, 1..5: user rating
	Rating = "Rating"

	Make  = "Make"  // recording equipment manufacturer name
	Model = "Model" // recording equipment model name or number
)

type Metadata struct {
	// Attr lists metadata attributes
	Attr map[string]string
}

func (m *Metadata) Set(name, value string) {
	if m.Attr == nil {
		m.Attr = make(map[string]string)
	}
	m.Attr[name] = value
}

var ErrUnknownFormat = errors.New("metadata: unknown content format")
var ErrNoMeta = errors.New("metadata: no metadata found")

const sniffLen = 256

// Parse parses metadata from r. Currently EXIF and MP4 headers are supported.
// If r is also an io.Seeker, then it is used to seek within r.
func Parse(r io.Reader) (*Metadata, error) {
	p := make([]byte, sniffLen)
	n, err := io.ReadFull(r, p)
	switch err {
	case io.ErrUnexpectedEOF:
		err = io.EOF
	case io.EOF:
		// pass
	default:
		return nil, err
	}

	return parse(p[:n], prefixReader(p, r))
}

func ParseAt(r io.ReaderAt) (*Metadata, error) {
	p := make([]byte, sniffLen)
	n, err := r.ReadAt(p, 0)
	switch err {
	case io.ErrUnexpectedEOF:
		err = io.EOF
	case io.EOF:
		// pass
	default:
		return nil, err
	}

	return parse(p[:n], &atReadSeeker{0, r})
}

func parse(p []byte, r io.Reader) (*Metadata, error) {
	if isjpeg(p) {
		return parseJpeg(r)
	}
	if ismp4(p) {
		return parseMP4(r)
	}

	return nil, ErrUnknownFormat
}

func isjpeg(p []byte) bool {
	if len(p) < 3 {
		return false
	}
	return bytes.Equal(p[:3], []byte{0xff, 0xd8, 0xff})
}

func ismp4(p []byte) bool {
	if len(p) < 12 {
		return false
	}

	boxSize := int(binary.BigEndian.Uint32(p[:4]))
	if boxSize%4 != 0 {
		return false
	}

	if !bytes.Equal(p[4:8], []byte("ftyp")) {
		return false
	}

	// don't care about the acutal codec, not relevant for metadata
	return true
}
