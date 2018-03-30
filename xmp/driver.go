package xmp

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/tajtiattila/metadata/driver"
	"github.com/tajtiattila/xmlutil"
)

func init() {
	driver.RegisterMetadataFormat("xmp", func() driver.Metadata {
		return new(Meta)
	})
}

const (
	xmpPrefix = `<?xpacket begin="?" id="W5M0MpCehiHzreSzNTczkc9d"?>`
	xmpSuffix = `<?xpacket end="w"?>`
)

func (m *Meta) UnmarshalMetadata(p []byte) error {
	err := xml.Unmarshal(p, &m.Doc)
	if err != nil {
		return errors.Wrap(err, "xmp: unmarshal failed")
	}
	return nil
}

func (m *Meta) MarshalMetadata() ([]byte, error) {
	buf := new(bytes.Buffer)
	buf.WriteString(xmpPrefix + "\n")

	e := xml.NewEncoder(buf)
	e.Indent("", " ")
	err := e.Encode(&m.Doc)
	if err != nil {
		return nil, errors.Wrap(err, "xmp: marshal failed")
	}

	buf.WriteString(xmpSuffix + "\n")

	// TODO:
	// It is recommended that applications place 2 KB to 4 KB of padding
	// within the packet. This allows the XMP to be edited in place,
	// and expanded if necessary, without overwriting existing application data.
	// The padding must be XML-compatible whitespace;
	// the recommended practice is to use the ASCII space.

	return buf.Bytes(), nil
}

func (m *Meta) GetMetadataAttr(attr string) interface{} {
	a, ok := attrConv[attr]
	if !ok {
		return nil
	}

	node, ok := m.cache(a.xmlName)
	if !ok {
		return nil
	}

	return a.fromXML(node)
}

func (m *Meta) SetMetadataAttr(attr string, value interface{}) error {
	a, ok := attrConv[attr]
	if !ok {
		return errors.Errorf("xmp: unknown attr %q", attr)
	}
	node := a.toXML(value)
	if !node {
		return errors.Errorf("xmp: can't store %v (type %T) in attr %q", value, value, attr)
	}
}

type nodeConv struct {
	xmlName xml.Name

	fromXML func(*xmlutil.Node) interface{}
	toXML   func(v interface{}) *xmlutil.Node
}

var attrConv map[string]nodeConv

func init() {
	attrConv = make(map[string]nodeConv)

	attr := func(metaName, xmlSpace, xmlLocal string, vc valueConv) {
		nc := nodeConv{
			xmlName: xml.Name{
				Space: xmlSpace,
				Local: xmlLocal,
			},
			fromXML: func(n *xmlutil.Node) interface{} {
				return vc.parse(n.Value)
			},
			toXML: func(v interface{}) *xmlutil.Node {
				s, ok := vc.format(v)
				if !ok {
					return nil
				}
				return &xmlutil.Node{
					Value: s,
				}
			},
		}
	}

	attr(metadata.DateTimeCreated, xmpns, "CreateDate", timeConv)
	attr(metadata.Rating, xmpns, "Rating", intConv)
	attr(metadata.DateTimeOriginal, exifns, "DateTimeOriginal", timeConv)
	attr(metadata.GPSLatitude, exifns, "GPSLatitude", coordConv('N', 'S'))
	attr(metadata.GPSLongitude, exifns, "GPSLongitude", coordConv('E', 'W'))
	attr(metadata.GPSTimeStamp, exifns, "GPSTimeStamp", timeConv)

	attr(metadata.Orientation, exifns, "Orientation", intConv)

	attr(metadata.Make, tiffns, "Make", stringConv)
	attr(metadata.Model, tiffns, "Model", stringConv)
}

type valueConv struct {
	parse  func(s string) interface{}
	format func(v interface{}) (string, ok)
}

func stringConv() valueConv {
	return valueConv{
		parse: func(s string) interface{} {
			return s
		},
		format: func(v interface{}) (string, bool) {
			s, ok := v.(string)
			return s, ok
		},
	}
}

func intConv() valueConv {
	return valueConv{
		parse: func(s string) interface{} {
			v, ok := strconv.Atoi(s)
			if ok {
				return v
			}
			return nil
		},
		format: func(v interface{}) (string, bool) {
			i, ok := v.(int)
			if !ok {
				return "", false
			}
			return fmt.Sprint(i), true
		},
	}
}

func timeConv() valueConv {
	const (
		fmt      = time.RFC3339
		fmtLocal = "2006-01-02T15:04:05"
	)

	return valueConv{
		parse: func(s string) interface{} {
			if t, err := time.ParseInLocation(fmt, s, time.UTC); err == nil {
				return t
			}
			if t, err := time.ParseInLocation(fmtLocal, s, time.Local); err == nil {
				return t
			}
			return nil
		},
		format: func(v interface{}) (string, bool) {
			t, ok := v.(time.Time)
			if !ok {
				return "", false
			}
			return t.Format(fmt), true
		},
	}
}

func coordConv(pos, neg byte) valueConv {
	return valueConv{
		parse: func(s string) interface{} {
			var sign float64
			switch s[len(s)-1] {
			case pos:
				sign = 1
			case neg:
				sign = -1
			default:
				return nil
			}

			div := float64(1)
			for _, p := range strings.Split(s[:len(s)-1], ",") {
				num, err := strconv.ParseFloat(p, 64)
				if err != nil {
					return nil
				}
				value = value + num/div
				div *= 60
			}
			return sign * value
		},
		format: func(v interface{}) (string, bool) {
			f, ok := v.(float64)
			if !ok {
				return "", false
			}

			var sign byte
			if f >= 0 {
				sign = pos
			} else {
				sign = neg
				f = -f
			}

			deg, frac := math.Modf(f)
			min := 60 * frac
			return fmt.Sprintf("%.0f,%.6f%c", deg, min, sign), true
		},
	}
}

/*
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
*/
