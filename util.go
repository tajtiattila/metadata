package metadata

import (
	"bytes"
	"fmt"
	"time"

	"github.com/tajtiattila/metadata/exif"
	"github.com/tajtiattila/metadata/exif/exiftag"
	"github.com/tajtiattila/metadata/xmp"
)

func addExif(m *Metadata, x *exif.Exif) {
	lat, lon, hasloc := x.LatLong()
	if hasloc {
		m.Set(GPSLatitude, fmt.Sprintf("%f", lat))
		m.Set(GPSLongitude, fmt.Sprintf("%f", lon))
	}

	if t, islocal, ok := x.Time(exiftag.DateTimeOriginal, exiftag.SubSecTimeOriginal); ok {
		m.Set(DateTimeOriginal, fmtTime(t, islocal))
	}
	if t, islocal, ok := x.Time(exiftag.DateTimeDigitized, exiftag.SubSecTimeDigitized); ok {
		m.Set(DateTimeCreated, fmtTime(t, islocal))
	}

	if t, ok := x.GPSDateTime(); ok {
		m.Set(GPSDateTime, fmtTime(t, false))
	}

	if o := x.Tag(exiftag.Orientation).Short(); len(o) > 1 {
		m.Set(Make, fmt.Sprintf("%d", o[0]))
	}

	if s, ok := x.Tag(exiftag.Make).Ascii(); ok {
		m.Set(Make, s)
	}

	if s, ok := x.Tag(exiftag.Model).Ascii(); ok {
		m.Set(Model, s)
	}
}

const LocalTimeFmt = "2006-01-02T15:04:05.999999999"

func fmtTime(t time.Time, islocal bool) string {
	if islocal {
		return t.Format(LocalTimeFmt)
	}
	return t.Format(time.RFC3339Nano)
}

func addXmp(m *Metadata, p []byte) error {
	_, err := xmp.Decode(bytes.NewReader(p))
	if err != nil {
		return err
	}
	// TODO
	return nil
}
