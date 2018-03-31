// Package metadata parses metadata in media files.
package metadata

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"time"
)

// Metadata records file metadata.
type Metadata struct {
	// Date of original image (eg. scanned photo)
	DateTimeOriginal time.Time

	// Original file creation date (eg. time of scan)
	DateTimeCreated time.Time

	// GPS records GPS information.
	GPS struct {
		GPSInfo

		// Valid indicates if Latitude and Longitude
		// fields of GPSInfo are valid.
		Valid bool
	}

	// Orientation is the Exif orientation.
	// Possible values are based on the exif spec:
	//   0: undefined
	//   1: no rotation
	//   2: flip horizontal
	//   3: rotate 180°
	//   4: flip vertical
	//   5: transpose
	//   6: rotate 90°
	//   7: transverse
	//   8: rotate 270°
	Orientation int

	// Rating is the XMP rating. Possible values are:
	//  -1: rejected
	//   0: unrated or missing
	//   1..5: user rating
	Rating int

	// Recording equipment manufacturer and model name/number name
	Make, Model string

	// Attr holds metadata attributes. Values are one of these types:
	//   time.Time
	//   string
	//   int
	//   float64
	Attr map[string]interface{}
}

// GPSInfo records GPS information.
type GPSInfo struct {
	// Latitude and Longitude are the geographical location.
	// Positive latitude means north, positive longitude means east.
	Latitude  float64
	Longitude float64

	// Time is time of the GPS fix. Zero means undefined.
	Time time.Time
}

// Attribute names read from media files.
//
// Date/time values are formatted as expected by
// the Time type of this package.
const (
	// date of original image (eg. scanned photo)
	DateTimeOriginal = "DateTimeOriginal"

	// original file creation date (eg. time of scan)
	DateTimeCreated = "DateTimeCreated"

	// Date/time of GPS fix (RFC3339, always UTC)
	GPSDateTime = "GPSDateTime"

	// latitude and longitude are signed floating point
	// values formatted with no exponent
	GPSLatitude  = "GPSLatitude"  // +north, -south
	GPSLongitude = "GPSLongitude" // +east, -west

	// Orientation (integer) 1..8, values are like exif
	Orientation = "Orientation"

	// XMP Rating (integer), -1: rejected, 0: unrated, 1..5: user rating
	Rating = "Rating"

	// recording equipment manufacturer and model name/number name
	Make  = "Make"
	Model = "Model"
)

// Set sets a metadata attribute.
func (m *Metadata) Set(key string, value interface{}) {
	if m.Attr == nil {
		m.Attr = make(map[string]interface{})
	}
	m.Attr[key] = value

	if f, ok := updateValue[key]; ok {
		f(m, value)
	}
}

// Get returns a metadata attribute.
func (m *Metadata) Get(key string) interface{} {
	return m.Attr[key]
}

// ErrUnknownFormat is returned by Parse and ParseAt when the file format
// is not understood by this package.
var ErrUnknownFormat = driver.ErrUnknownFormat

// ErrNoMeta is returned by Parse and ParseAt when the file format
// was recognised but no metadata was found.
var ErrNoMeta = errors.New("metadata: no metadata found")

const sniffLen = 256

// Parse parses metadata from r, and returns the metadata found
// and the first error encountered.
//
// Metadata is parsed on a best effort basis.
// Valid values are always returned
// even when if non-fatal errors had been encountered by decoding
// the underlying formats.
//
// If r is also an io.Seeker, then it is used to seek within r.
func Parse(r io.Reader) (*Metadata, error) {
	p := make([]byte, sniffLen)
	n, err := io.ReadFull(r, p)
	switch err {
	case io.ErrUnexpectedEOF:
		err = io.EOF
	case io.EOF, nil:
		// pass
	default:
		return nil, err
	}

	return parse(p[:n], prefixReader(p, r))
}

// ParseAt parses metadata from r, and returns the metadata found
// and the first error encountered.
//
// Metadata is parsed on a best effort basis.
// Valid values are always returned
// even when if non-fatal errors had been encountered by decoding
// the underlying formats.
func ParseAt(r io.ReaderAt) (*Metadata, error) {
	return Parse(&atReadSeeker{0, r})
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

// TimeAttrs lists time attributes recognised in Merge.
var TimeAttrs = setOf(DateTimeOriginal, DateTimeCreated, GPSDateTime)

// Merge merges metadata from multiple sources.
func Merge(v ...*Metadata) *Metadata {
	switch len(v) {
	case 0:
		return nil
	case 1:
		return v[0]
	}

	result := new(Metadata)
	for _, m := range v {
		for key, val := range m.Attr {
			if _, ok := TimeAttrs[key]; ok {
				r, ok := result.Attr[key]
				if !ok || timeBetter(val, r) {
					result.Set(key, val)
				}
			} else {
				result.Set(key, val)
			}
		}
	}
	return result
}

func timeBetter(val, than interface{}) bool {
	tv, ok := val.(time.Time)
	if !ok {
		return true
	}
	vallocal := tv.Location() == time.Local

	tt, ok := than.(time.Time)
	if !ok {
		return false
	}
	thanlocal := tt.Location() == time.Local

	return vallocal && !thanlocal
}

func setOf(v ...string) map[string]struct{} {
	m := make(map[string]struct{})
	for _, s := range v {
		m[s] = struct{}{}
	}
	return m
}

type updateFunc func(m *Metadata, value interface{})

var updateValue = map[string]updateFunc{
	DateTimeOriginal: func(m *Metadata, v interface{}) { updateTime(&m.DateTimeOriginal, v) },
	DateTimeCreated:  func(m *Metadata, v interface{}) { updateTime(&m.DateTimeCreated, v) },
	GPSDateTime:      func(m *Metadata, v interface{}) { updateTime(&m.GPS.Time, v) },
	GPSLatitude:      updateLatLong,
	GPSLongitude:     updateLatLong,
	Orientation:      func(m *Metadata, v interface{}) { updateInt(&m.Orientation, v) },
	Rating:           func(m *Metadata, v interface{}) { updateInt(&m.Rating, v) },
	Make:             func(m *Metadata, v interface{}) { updateString(&m.Make, v) },
	Model:            func(m *Metadata, v interface{}) { updateString(&m.Model, v) },
}

func updateString(p *string, v interface{}) {
	if s, ok := v.(string); ok {
		*p = s
	}
}

func updateTime(p *time.Time, v interface{}) {
	if t, ok := v.(time.Time); ok {
		*p = t
	}
}

func updateInt(p *int, v interface{}) {
	if i, ok := v.(int); ok {
		*p = i
	}
}

func updateLatLong(m *Metadata, v interface{}) {
	var latok, lonok bool
	m.GPS.Latitude, latok = m.Get(GPSLatitude).(float64)
	m.GPS.Longitude, lonok = m.Get(GPSLongitude).(float64)
	m.GPS.Valid = latok && lonok
}
