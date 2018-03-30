package metadata

/*
import (
	"bytes"
	"fmt"

	"github.com/tajtiattila/metadata/xmp"
)

func FromXMPBytes(p []byte) (*Metadata, error) {
	x, err := xmp.Decode(bytes.NewReader(p))
	if err != nil {
		return nil, err
	}
	return FromXMP(x), nil
}

// FromXMP decodes XMP metadata.
func FromXMP(x *xmp.Meta) *Metadata {
	m := new(Metadata)
	for _, a := range xmpAttr {
		if v, ok := a.getf(x); ok {
			m.Set(a.metaName, v)
		}
	}
	return m
}

var xmpAttr = []struct {
	metaName string
	getf     func(x *xmp.Meta) (string, bool)
}{
	{DateTimeCreated, xmpString(xmp.CreateDate)},
	{DateTimeOriginal, xmpString(xmp.DateTimeOriginal)},
	{GPSDateTime, xmpString(xmp.GPSTimeStamp)},

	{Rating, xmpInt(xmp.Rating)},

	{GPSLatitude, xmpFloat(xmp.GPSLatitude)},
	{GPSLongitude, xmpFloat(xmp.GPSLongitude)},

	{Orientation, xmpInt(xmp.Orientation)},

	{Make, xmpString(xmp.Make)},
	{Model, xmpString(xmp.Model)},
}

func xmpString(a xmp.StringFunc) func(x *xmp.Meta) (string, bool) {
	return func(x *xmp.Meta) (string, bool) {
		return x.String(a)
	}
}

func xmpFloat(a xmp.Float64Func) func(x *xmp.Meta) (string, bool) {
	return func(x *xmp.Meta) (string, bool) {
		f, ok := x.Float64(a)
		if ok {
			return fmt.Sprintf("%v", f), true
		}
		return "", false
	}
}

func xmpInt(a xmp.IntFunc) func(x *xmp.Meta) (string, bool) {
	return func(x *xmp.Meta) (string, bool) {
		i, ok := x.Int(a)
		if ok {
			return fmt.Sprintf("%d", i), true
		}
		return "", false
	}
}
*/
