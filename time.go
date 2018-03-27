package metadata

import "time"

func markLocal(t time.Time) time.Time {
	return t.Local()
}

func markNonLocal(t time.Time) time.Time {
	if t.Location() != time.Local {
		return t
	}

	_, off := t.Zone()
	return t.In(time.FixedZone("", off))
}

// FixLocalTime returns the time t in loc if t.Location() == time.Local
// but keeps the original date and clock values.
// In other words, the source and result times
// may differ only in their time zone parts when formatted.
//
// It can be used to correct time values from metadata formats
// such as Exif that support local time only.
//
// FixLocalTime panics if t is nil.
// It has no effect if loc == time.Local.
func FixLocalTime(t time.Time, loc *time.Location) time.Time {
	if t.Location() != time.Local {
		return t
	}
	if loc == time.Local {
		return t
	}

	_, off0 := t.Zone()
	t = t.In(loc)
	_, off1 := t.Zone()

	return t.Add(time.Duration(off0-off1) * time.Second)
}
