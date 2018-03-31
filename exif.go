package metadata

/*
import (
	"github.com/tajtiattila/metadata/exif"
	"github.com/tajtiattila/metadata/exif/exiftag"
)

func FromExifBytes(p []byte) (*Metadata, error) {
	x, err := exif.DecodeBytes(p)
	if x != nil {
		m := FromExif(x)
		if m != nil && len(m.Attr) != 0 {
			return m, err
		}
	}
	return nil, err
}

func FromExif(x *exif.Exif) *Metadata {
	m := new(Metadata)

	if i, ok := x.GPSInfo(); ok {
		m.Set(GPSLatitude, i.Lat)
		m.Set(GPSLongitude, i.Long)
		if !i.Time.IsZero() {
			m.Set(GPSDateTime, i.Time)
		}
	}

	if t, ok := x.Time(exiftag.DateTimeOriginal, exiftag.SubSecTimeOriginal); ok {
		m.Set(DateTimeOriginal, t)
	}
	if t, ok := x.Time(exiftag.DateTimeDigitized, exiftag.SubSecTimeDigitized); ok {
		m.Set(DateTimeCreated, t)
	}

	if o := x.Tag(exiftag.Orientation).Short(); len(o) > 0 {
		m.Set(Orientation, o[0])
	}

	if s, ok := x.Tag(exiftag.Make).Ascii(); ok {
		m.Set(Make, s)
	}

	if s, ok := x.Tag(exiftag.Model).Ascii(); ok {
		m.Set(Model, s)
	}
	return m
}
*/
