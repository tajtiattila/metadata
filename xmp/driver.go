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
	"github.com/tajtiattila/metadata"
	"github.com/tajtiattila/metadata/metaio"
	"github.com/tajtiattila/xmlutil"
)

func init() {
	metaio.RegisterMetadataFormat("xmp", func(...metaio.Option) metaio.Metadata {
		return new(Meta)
	})
}

const (
	xmpPrefix = `<?xpacket begin="?" id="W5M0MpCehiHzreSzNTczkc9d"?>`
	xmpSuffix = `<?xpacket end="w"?>`

	toolkit = "github.com/tajtiattila/metadata/xmp"
)

var _ metaio.Metadata = new(Meta)

func (m *Meta) MetadataName() string { return "xmp" }

func (m *Meta) UnmarshalMetadata(p []byte) error {
	var doc xmlutil.Document
	err := xml.Unmarshal(p, &doc)
	if err != nil {
		return errors.Wrap(err, "xmp: unmarshal failed")
	}
	m.Doc = doc
	return nil
}

func (m *Meta) MarshalMetadata() ([]byte, error) {
	if m.Doc.Node == nil {
		return nil, errors.New("xmp: empty document")
	}

	if m.Doc.Node.Name == xmlName(rdfns, "RDF") {
		// add xmpmeta
		xm := &xmlutil.Node{
			Name: xmlName(metans, "xmpmeta"),
			Attr: xattr(
				"xmlns", "x", metans,
				metans, "xmptk", toolkit,
			),
			Child: []*xmlutil.Node{m.Doc.Node},
		}
		m.Doc.Node = xm
	}

	buf := new(bytes.Buffer)
	buf.WriteString(xmpPrefix + "\n")

	e := xml.NewEncoder(buf)
	e.Indent("", " ")
	err := e.Encode(&m.Doc)
	if err != nil {
		return nil, errors.Wrap(err, "xmp: marshal failed")
	}

	buf.WriteString("\n" + xmpSuffix + "\n")

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

	node, ok := m.attr[a.xmlName]
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
	if node == nil {
		return errors.Errorf("xmp: can't store %v (type %T) in attr %q", value, value, attr)
	}
	node.Name = a.xmlName

	descr := m.getDescr(a.xmlName.Space)
	if descr == nil {
		return errors.Errorf("xmp: unknown namespace for attr %q", attr)
	}

	for i, n := range descr.Child {
		if n.Name == a.xmlName {
			// replace existing node
			descr.Child[i] = node
			return nil
		}
	}

	// add new node
	descr.Child = append(descr.Child, node)

	if m.attr == nil {
		m.attr = make(map[xml.Name]*xmlutil.Node)
	}
	m.attr[a.xmlName] = node

	return nil
}

func (m *Meta) DeleteMetadataAttr(attr string) error {
	a, ok := attrConv[attr]
	if !ok {
		return errors.Errorf("xmp: unknown attr %q", attr)
	}

	for _, descr := range m.descr {
		for i := 0; i < len(descr.Child); {
			if descr.Child[i].Name == a.xmlName {
				descr.Child = append(descr.Child[:i], descr.Child[i+1:]...)
			} else {
				i++
			}
		}
	}

	if m.attr != nil {
		delete(m.attr, a.xmlName)
	}
	return nil
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
		attrConv[metaName] = nc
	}

	attr(metadata.DateTimeCreated, xmpns, "CreateDate", timeConv)
	attr(metadata.Rating, xmpns, "Rating", intConv)
	attr(metadata.DateTimeOriginal, exifns, "DateTimeOriginal", timeConv)
	attr(metadata.GPSLatitude, exifns, "GPSLatitude", coordConv('N', 'S'))
	attr(metadata.GPSLongitude, exifns, "GPSLongitude", coordConv('E', 'W'))
	attr(metadata.GPSDateTime, exifns, "GPSTimeStamp", timeConv)

	attr(metadata.Orientation, exifns, "Orientation", intConv)

	attr(metadata.Make, tiffns, "Make", stringConv)
	attr(metadata.Model, tiffns, "Model", stringConv)
}

type valueConv struct {
	parse  func(s string) interface{}
	format func(v interface{}) (string, bool)
}

var stringConv = valueConv{
	parse: func(s string) interface{} {
		return s
	},
	format: func(v interface{}) (string, bool) {
		s, ok := v.(string)
		return s, ok
	},
}

var intConv = valueConv{
	parse: func(s string) interface{} {
		v, err := strconv.Atoi(s)
		if err == nil {
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

const (
	timeFmt      = time.RFC3339
	timeFmtLocal = "2006-01-02T15:04:05"
)

var timeConv = valueConv{
	parse: func(s string) interface{} {
		if t, err := time.ParseInLocation(timeFmt, s, time.UTC); err == nil {
			return t
		}
		if t, err := time.ParseInLocation(timeFmtLocal, s, time.Local); err == nil {
			return t
		}
		return nil
	},
	format: func(v interface{}) (string, bool) {
		t, ok := v.(time.Time)
		if !ok {
			return "", false
		}
		return t.Format(timeFmt), true
	},
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

			var value float64
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
