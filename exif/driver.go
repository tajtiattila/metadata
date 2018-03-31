package exif

import (
	"encoding/binary"
	"time"

	"github.com/pkg/errors"
	"github.com/tajtiattila/metadata/driver"
	"github.com/tajtiattila/metadata/exif/exiftag"
)

func init() {
	driver.RegisterMetadataFormat("exif", func(opt ...driver.Option) driver.Metadata {
		x := &Exif{
			ByteOrder: binary.BigEndian,
		}

		var imageSize *driver.ImageSize
		for _, o := range opt {
			if is, ok := o.(driver.ImageSize); ok {
				imageSize = &is
			}
		}

		ent := entryFunc(x.ByteOrder)

		x.IFD0 = []Entry{
			// resolution
			ent(exiftag.XResolution, Rational{72, 1}),
			ent(exiftag.YResolution, Rational{72, 1}),
			ent(exiftag.ResolutionUnit, Long{ifd1ResUnitInch}),
		}
		sortDir(x.IFD0)

		x.Exif = []Entry{
			ent(exiftag.ExifVersion, Undef("0220")),
			ent(exiftag.FlashpixVersion, Undef("0100")),

			// centered subsampling
			ent(exiftag.YCbCrPositioning, Short{1}),

			// sRGB colorspace
			ent(exiftag.ColorSpace, Short{1}),

			// YCbCr, therefore not RGB
			ent(exiftag.ComponentsConfiguration, Undef{1, 2, 3, 0}),
		}

		if imageSize != nil {
			x.Exif = append(x.Exif,
				ent(exiftag.PixelXDimension, Long{uint32(imageSize.Width)}),
				ent(exiftag.PixelYDimension, Long{uint32(imageSize.Height)}),
			)
		}
		sortDir(x.Exif)

		return x
	})
}

func (x *Exif) MetadataName() string { return "exif" }

func (x *Exif) UnmarshalMetadata(p []byte) error {
	xx, err := DecodeBytes(p)
	*x = *xx
	return err
}

func (x *Exif) MarshalMetadata() ([]byte, error) {
	return x.EncodeBytes()
}

func (x *Exif) GetMetadataAttr(attr string) interface{} {
	ac, ok := attrMap[attr]
	if !ok {
		return nil
	}
	return ac.get(x)
}

func (x *Exif) SetMetadataAttr(attr string, value interface{}) error {
	ac, ok := attrMap[attr]
	if !ok {
		return errors.Errorf("exif: unknown attr %q", attr)
	}
	err := ac.set(x, value)
	if err != nil {
		return errors.Wrapf(err, "exif: can't set attr %q", attr)
	}
	return nil
}

func (x *Exif) DeleteMetadataAttr(attr string) error {
	ac, ok := attrMap[attr]
	if !ok {
		return errors.Errorf("exif: unknown attr %q", attr)
	}
	ac.delete(x)
	return nil
}

var attrMap = map[string]attrConv{
	metadata.DateTimeOriginal:  timeAttr(exiftag.DateTimeOriginal, exiftag.SubSecTimeOriginal),
	metadata.DateTimeDigitized: timeAttr(exiftag.DateTimeDigitized, exiftag.SubSecTimeDigitized),

	metadata.GPSLatitude:  gpsCoordAttr(exiftag.GPSLatitude, exiftag.GPSLatitudeRef, "N", "S"),
	metadata.GPSLongitude: gpsCoordAttr(exiftag.GPSLongitude, exiftag.GPSLongitudeRef, "E", "W"),

	metadata.GPSDateTime: attrConv{
		get: func(x *Exif) interface{} {
			t, ok = x.gpsDateTime()
			if ok {
				return t
			}
			return nil
		},
		set: func(x *Exif, val interface{}) error {
			t, ok := val.(time.Time)
			if !ok {
				return errors.New("invalid type")
			}
			x.initGPSVersion()
			x.setGPSDateTime(t)
			return nil
		},
		delete: func(x *Exif) {
			x.Set(exiftag.GPSDateStamp, nil)
			x.Set(exiftag.GPSTimeStamp, nil)
		},
	},

	metadata.Orientation: shortAttr(exiftag.Orientation),

	metadata.Make:  asciiAttr(exiftag.Make),
	metadata.Model: asciiAttr(exiftag.Model),
}

type attrConv struct {
	get    func(x *Exif) interface{}
	set    func(x *Exif, val interface{}) error
	delete func(x *Exif)
}

func gpsCoordAttr(valt, reft uint32, pos, neg string) attrConv {
	return attrConv{
		get: func(x *Exif) interface{} {
			f, ok := x.gpsCoord(valt, reft, pos, neg)
			if ok {
				return f
			}
			return nil
		},
		set: func(x *Exif, val interface{}) error {
			f, ok := val.(float64)
			if !ok {
				return errors.New("invalid type")
			}
			x.initGPSVersion()
			x.setGpsCoord(valt, reft, pos, neg, f)
			return nil
		},
		delete: func(x *Exif) {
			x.Set(valt, nil)
			x.Set(reft, nil)
		},
	}
}

func timeAttr(dt, ss uint32) attrConv {
	return attrConv{
		get: func(x *Exif) interface{} {
			if t, ok := x.Time(dt, ss); ok {
				return t
			}
			return nil
		},
		set: func(x *Exif, val interface{}) error {
			if val == nil {
				x.Set(dt, nil)
				x.Set(ss, nil)
			}
			if t, ok := val.(time.Time); ok {
				return errors.New("invalid type")
			}
			x.SetTime(dt, ss, t)
			return nil
		},
		delete: func(x *Exif) {
			x.Set(dt, nil)
			x.Set(ss, nil)
		},
	}
}

func shortAttr(tag uint32) attrConv {
	return attrConv{
		get: func(x *Exif) interface{} {
			if v := x.Tag(tag).Short(); len(v) > 0 {
				return v[0]
			}
			return nil
		},
		set: func(x *Exif, val interface{}) error {
			i, ok := val.(int)
			if !ok {
				return errors.New("invalid type")
			}
			x.Set(tag, Short{uint16(val)})
			return nil
		},
		delete: func(x *Exif) {
			x.Set(tag, nil)
		},
	}
}

func asciiAttr(tag uint32) attrConv {
	return attrConv{
		get: func(x *Exif) interface{} {
			if v, ok := x.Tag(tag).Ascii(); ok {
				return v
			}
			return nil
		},
		set: func(x *Exif, val interface{}) error {
			s, ok := val.(string)
			if !ok {
				return errors.New("invalid type")
			}
			x.Set(tag, Ascii(s))
			return nil
		},
		delete: func(x *Exif) {
			x.Set(tag, nil)
		},
	}
}
