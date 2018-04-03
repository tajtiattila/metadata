// Package metadata parses metadata in media files.
package metadata

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"time"

	"github.com/tajtiattila/metadata/metaio"
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
	// image or frame dimensinos [int]
	ImageWidth  = "ImageWidth"
	ImageHeight = "ImageHeight"

	// date of original image (eg. scanned photo) [time.Time]
	DateTimeOriginal = "DateTimeOriginal"

	// original file creation date (eg. time of scan) [time.Time]
	DateTimeCreated = "DateTimeCreated"

	// Date/time of GPS fix (RFC3339, always UTC) [time.Time]
	GPSDateTime = "GPSDateTime"

	// latitude and longitude are signed floating point
	// values formatted with no exponent [float64]
	GPSLatitude  = "GPSLatitude"  // +north, -south
	GPSLongitude = "GPSLongitude" // +east, -west

	// Orientation [int] 1..8, values are like exif
	Orientation = "Orientation"

	// XMP Rating [int], -1: rejected, 0: unrated, 1..5: user rating
	Rating = "Rating"

	// recording equipment manufacturer and model name/number name [string]
	Make  = "Make"
	Model = "Model"
)

var attrNames = []string{
	ImageWidth, ImageHeight,

	DateTimeOriginal, DateTimeCreated,

	GPSDateTime, GPSLatitude, GPSLongitude,

	Orientation,

	Rating,

	Make, Model,
}

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
var ErrUnknownFormat = metaio.ErrUnknownFormat

// ErrNoMeta is returned by Parse and ParseAt when the file format
// was recognised but no metadata was found.
var ErrNoMeta = errors.New("metadata: no metadata found")

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
	sniffLen := metaio.ContainerPeekLen()
	if sniffLen == 0 {
		// no registered format
		return nil, ErrUnknownFormat
	}

	p := make([]byte, sniffLen)

	r, err := peek(r, p)
	if err != nil {
		return nil, ErrUnknownFormat
	}

	cf, _ := metaio.GetContainerFormat(p)
	if cf == nil {
		return nil, ErrUnknownFormat
	}

	var metaErr error
	result := new(Metadata)
	_, err = cf.Scan(r, func(name string, data []byte) {
		m := metaio.NewMetadata(name)
		if m == nil {
			return
		}

		if err := m.UnmarshalMetadata(data); err != nil {
			if metaErr == nil {
				metaErr = err
			}
			return
		}

		for _, n := range attrNames {
			result.update(n, m.GetMetadataAttr(n))
		}
	})

	if err != nil {
		return result, err
	}

	return result, metaErr
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

func peek(r io.Reader, p []byte) (io.Reader, error) {
	if ra, ok := r.(io.ReaderAt); ok {
		_, err := ra.ReadAt(p, 0)
		if err != nil {
			return nil, err
		}
		return r, nil
	}

	if _, err := io.ReadFull(r, p); err != nil {
		return nil, ErrUnknownFormat
	}

	if rs, ok := r.(io.ReadSeeker); ok {
		if _, err := rs.Seek(0, io.SeekStart); err == nil {
			return r, nil
		}
		// seeker can't seek
	}

	buf := make([]byte, len(p))
	copy(buf, p)
	return io.MultiReader(bytes.NewReader(buf), r), nil
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

func (m *Metadata) update(attr string, value interface{}) {
	if value == nil {
		return
	}
	if t, ok := value.(time.Time); ok {
		if existing, ok := m.Attr[attr]; ok && timeBetter(existing, t) {
			return
		}
	}
	m.Set(attr, value)
}

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
			if t, ok := val.(time.Time); ok {
				existing, ok := result.Attr[key]
				if ok || timeBetter(existing, t) {
					continue
				}
			}
			result.Set(key, val)
		}
	}
	return result
}

func timeBetter(val interface{}, than time.Time) bool {
	tv, ok := val.(time.Time)
	if !ok {
		return true
	}
	vallocal := tv.Location() == time.Local

	thanlocal := than.Location() == time.Local

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
