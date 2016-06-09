package exif

import (
	"encoding/binary"
	"math"
	"time"

	"github.com/tajtiattila/exif-go/exif/exiftag"
)

// New initializes a new Exif structure for an image
// with the provided dimensions.
func New(dx, dy int) *Exif {
	bo := binary.BigEndian
	x := &Exif{ByteOrder: bo}

	ent := entryFunc(x.ByteOrder)

	x.IFD0 = Dir{
		// resolution
		ent(exiftag.XResolution, Rational{72, 1}),
		ent(exiftag.YResolution, Rational{72, 1}),
		ent(exiftag.ResolutionUnit, Long{ifd1ResUnitInch}),
	}
	x.IFD0.Sort()

	x.Exif = Dir{
		ent(exiftag.ExifVersion, Undef("0220")),
		ent(exiftag.FlashpixVersion, Undef("0100")),

		ent(exiftag.PixelXDimension, Long{uint32(dx)}),
		ent(exiftag.PixelYDimension, Long{uint32(dx)}),

		// centered subsampling
		ent(exiftag.YCbCrPositioning, Short{1}),

		// sRGB colorspace
		ent(exiftag.ColorSpace, Short{1}),

		// YCbCr, therefore not RGB
		ent(exiftag.ComponentsConfiguration, Undef{1, 2, 3, 0}),
	}

	return x
}

// Time reports the time from the specified DateTime and SubSecTime tags.
func (x *Exif) Time(timeTag, subSecTag uint32) (t time.Time, islocal, ok bool) {
	return timeFromTags(x.Tag(exiftag.DateTimeOriginal), x.Tag(exiftag.SubSecTimeOriginal))
}

// DateTime reports the Exif datetime. The fields checked
// in order are Exif/DateTimeOriginal, Exif/DateTimeDigitized and
// Tiff/DateTime. If neither is available, ok == false is returned.
func (x *Exif) DateTime() (t time.Time, ok bool) {
	t, _, ok = x.Time(exiftag.DateTimeOriginal, exiftag.SubSecTimeOriginal)
	if ok {
		return
	}

	t, _, ok = x.Time(exiftag.DateTimeDigitized, exiftag.SubSecTimeDigitized)
	if ok {
		return
	}

	t, _, ok = x.Time(exiftag.DateTime, exiftag.SubSecTime)
	return
}

// SetDateTime sets the fields
// Exif/DateTimeOriginal, Exif/DateTimeDigitized and
// Tiff/DateTime to t.
func (x *Exif) SetDateTime(t time.Time) {
	v, subv := timeValues(t)

	x.Set(exiftag.DateTimeOriginal, v)
	x.Set(exiftag.SubSecTimeOriginal, subv)

	x.Set(exiftag.DateTimeDigitized, v)
	x.Set(exiftag.SubSecTimeDigitized, subv)

	x.Set(exiftag.DateTime, v)
	x.Set(exiftag.SubSecTime, subv)
}

// LatLong reports the GPS latitude and longitude.
func (x *Exif) LatLong() (lat, long float64, ok bool) {
	latsig, ok1 := locSig(x.Tag(exiftag.GPSLatitudeRef), "N", "S")
	lonsig, ok2 := locSig(x.Tag(exiftag.GPSLongitudeRef), "E", "W")
	latabs, ok3 := degHourMin(x.Tag(exiftag.GPSLatitude))
	lonabs, ok4 := degHourMin(x.Tag(exiftag.GPSLongitude))
	if ok1 && ok2 && ok3 && ok4 {
		return latsig * latabs, lonsig * lonabs, true
	}

	return 0, 0, false
}

// SetLatLong sets the GPS latitude and longitude.
func (x *Exif) SetLatLong(lat, lon float64) {

	x.Set(exiftag.GPSVersionID, Byte{2, 2, 0, 0})
	var latsig string
	if lat < 0 {
		latsig = "S"
		lat = -lat
	} else {
		latsig = "N"
	}
	x.Set(exiftag.GPSLatitudeRef, Ascii(latsig))

	var lonsig string
	if lon < 0 {
		lonsig = "W"
		lon = -lon
	} else {
		lonsig = "E"
	}
	x.Set(exiftag.GPSLongitudeRef, Ascii(lonsig))

	x.Set(exiftag.GPSLatitude, toDegHourMin(lat))
	x.Set(exiftag.GPSLongitude, toDegHourMin(lon))
}

func (x *Exif) GPSDateTime() (t time.Time, ok bool) {
	ds, ok := x.Tag(exiftag.GPSDateStamp).Ascii()
	if !ok {
		return time.Time{}, false
	}

	d, err := time.Parse("2006:01:02", ds)
	if err != nil {
		return time.Time{}, false
	}

	thi, tlo, ok := x.Tag(exiftag.GPSTimeStamp).Rational().Sexagesimal(1e9)
	if !ok || thi != 0 {
		return time.Time{}, false
	}

	return d.Add(time.Duration(tlo) * time.Nanosecond), true
}

const TimeFormat = "2006:01:02T15:04:05"
const exTimeFormat = "2006:01:02T15:04:05Z"

func timeFromTags(t, subt *Tag) (tm time.Time, islocal, ok bool) {
	tm, islocal, ok = timePart(t)
	if !ok {
		return
	}

	subs, ok := subt.Ascii()
	if !ok {
		return tm, islocal, true
	}

	var nanos time.Duration
	res := time.Second
	for _, r := range subs {
		if '0' <= r && r <= '9' {
			nanos = nanos*10 + time.Duration(r-'0')
			res /= 10
			if res == 0 {
				break
			}
		} else {
			break
		}
	}
	return tm.Add(nanos * res), islocal, true
}

func timePart(t *Tag) (tm time.Time, islocal, ok bool) {
	tms, ok := t.Ascii()
	if !ok {
		return
	}

	tm, err := time.Parse(exTimeFormat, tms)
	if err == nil {
		return tm, false, true
	}

	tm, err = time.ParseInLocation(TimeFormat, tms, time.Local)
	if err != nil {
		return tm, true, true
	}

	// parse prefix
	tm, err = time.ParseInLocation(TimeFormat, tms[:len(TimeFormat)], time.Local)
	if err == nil {
		return tm, true, true
	}

	return time.Time{}, false, false
}

func timeValues(t time.Time) (v, subv Value) {
	v = Ascii(t.Format(TimeFormat))

	nano := t.Nanosecond()
	if nano == 0 {
		return v, nil
	}

	p := make([]byte, 0, 10)
	res := int(1e8)
	for nano > 0 {
		digit := nano / res
		nano = nano % res
		res /= 10
		p = append(p, '0'+byte(digit))
	}
	subv = Ascii(p)
	return v, subv
}

func locSig(t *Tag, pos, neg string) (sig float64, ok bool) {
	s, ok := t.Ascii()
	if !ok {
		return 0, false
	}
	switch s {
	case pos:
		sig = 1
	case neg:
		sig = -1
	default:
		return 0, false
	}
	return sig, true
}

func degHourMin(t *Tag) (val float64, ok bool) {
	r := t.Rational()
	if len(r) != 6 {
		return 0, false
	}
	div := 1.0
	for i := 0; i < 3; i++ {
		num, denom := r[2*i], r[2*i+1]
		v := float64(num) / (div * float64(denom))
		val += v
		div *= 60
	}
	return val, true
}

func toDegHourMin(val float64) Rational {
	r := make([]uint32, 6)

	// whole degrees
	i, f := math.Modf(val)
	r[0] = uint32(i)
	r[1] = uint32(1)

	// whole minutes
	i, f = math.Modf(f * 60)
	r[2] = uint32(i)
	r[3] = uint32(1)

	// store lat/long fractions to 30 cm precision on equator
	const degreeFractions = 100

	f *= 60 * degreeFractions
	r[4] = uint32(f + 0.5)
	r[5] = degreeFractions

	if r[4] == 60*degreeFractions {
		r[4] = 0
		r[2]++
		if r[2] == 60 {
			r[2] = 0
			r[0]++
		}
	}

	return Rational(r)
}
