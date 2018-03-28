package xmp

import (
	"encoding/xml"
	"strconv"
	"strings"
)

const (
	xmpns  = "http://ns.adobe.com/xap/1.0/"
	exifns = "http://ns.adobe.com/exif/1.0/"
	tiffns = "http://ns.adobe.com/tiff/1.0/"
)

var (
	CreateDate = tagString(xmpns, "CreateDate") // used for exif/DateTimeDigitized

	Rating = tagInt(xmpns, "Rating")

	DateTimeOriginal = tagString(exifns, "DateTimeOriginal")

	GPSLatitude  = tagCoord(exifns, "GPSLatitude", 'N', 'S')
	GPSLongitude = tagCoord(exifns, "GPSLongitude", 'E', 'W')

	GPSTimeStamp = tagString(exifns, "GPSTimeStamp") // includes exif/GPSDateStamp

	Orientation = tagInt(exifns, "Orientation")

	Make  = tagString(tiffns, "Make")
	Model = tagString(tiffns, "Model")
)

type StringFunc func(m *Meta) (value string, ok bool)

type IntFunc func(m *Meta) (value int, ok bool)

type Float64Func func(m *Meta) (value float64, ok bool)

func tagString(space, local string) StringFunc {
	xn := xmlName(space, local)
	return func(m *Meta) (string, bool) {
		return findString(m, xn)
	}
}

func tagInt(space, local string) IntFunc {
	xn := xmlName(space, local)
	return func(m *Meta) (int, bool) {
		s, ok := findString(m, xn)
		if !ok {
			return 0, false
		}
		i, err := strconv.Atoi(s)
		return i, err == nil
	}
}

func tagCoord(space, local string, pos, neg byte) Float64Func {
	xn := xmlName(space, local)
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

func xmlName(space, local string) xml.Name {
	return xml.Name{
		Space: space,
		Local: local,
	}
}

func findString(m *Meta, name xml.Name) (s string, ok bool) {
	n, ok := m.cache[name]
	if !ok {
		return "", false
	}
	return n.Value, true
}
