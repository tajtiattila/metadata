package xmp

import (
	"encoding/xml"
	"strconv"
	"strings"
)

var nsmap = map[string]string{
	"xmp":    "http://ns.adobe.com/xap/1.0/",
	"tiff":   "http://ns.adobe.com/tiff/1.0/",
	"exif":   "http://ns.adobe.com/exif/1.0/",
	"exifex": "http://cipa.jp/exif/1.0/",
}

var (
	CreateDate = tagString("xmp:CreateDate") // used for exif/DateTimeDigitized

	Rating = tagInt("xmp:Rating")

	DateTimeOriginal = tagString("exif:DateTimeOriginal")

	GPSLatitude  = tagCoord("exif:GPSLatitude", 'N', 'S')
	GPSLongitude = tagCoord("exif:GPSLongitude", 'E', 'W')

	GPSTimeStamp = tagString("exif:GPSTimeStamp") // includes exif/GPSDateStamp

	Orientation = tagInt("exif:Orientation")

	Make  = tagString("tiff:Make")
	Model = tagString("tiff:Model")
)

type StringFunc func(m *Meta) (value string, ok bool)

type IntFunc func(m *Meta) (value int, ok bool)

type Float64Func func(m *Meta) (value float64, ok bool)

func tagString(name string) StringFunc {
	xn := xmlName(name)
	return func(m *Meta) (string, bool) {
		return findString(m, xn)
	}
}

func tagInt(name string) IntFunc {
	xn := xmlName(name)
	return func(m *Meta) (int, bool) {
		s, ok := findString(m, xn)
		if !ok {
			return 0, false
		}
		i, err := strconv.Atoi(s)
		return i, err == nil
	}
}

func tagCoord(name string, pos, neg byte) Float64Func {
	xn := xmlName(name)
	return func(m *Meta) (value float64, ok bool) {
		s, ok := findString(m, xn)
		if !ok || len(s) < 2 {
			return 0, false
		}
		var sign float64
		switch s[len(s)-1] {
		case pos:
			sign = 1
		case neg:
			sign = -1
		default:
			return 0, false
		}

		div := float64(1)
		for _, p := range strings.Split(s[:len(s)-1], ",") {
			num, err := strconv.ParseFloat(p, 64)
			if err != nil {
				return 0, false
			}
			value = value + num/div
			div *= 60
		}
		return sign * value, true
	}
}

func xmlName(t string) xml.Name {
	parts := strings.Split(t, ":")
	ns, ok := nsmap[parts[0]]
	if !ok {
		panic("invalid namespace")
	}
	return xml.Name{ns, parts[1]}
}

func findString(m *Meta, name xml.Name) (s string, ok bool) {
	n := findNode(m, name)
	if n != nil {
		return string(n.CharData), true
	}
	return "", false
}

func findNode(m *Meta, name xml.Name) *Node {
	for _, d := range m.Rdf.Desc {
		for i := range d.Node {
			n := &d.Node[i]
			if n.XMLName == name {
				return n
			}
		}
	}
	return nil
}
