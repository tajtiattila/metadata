package metadata

import "time"

// Time is like time.Time but records the precision
// (year, month, day, hour, minute, second or subsecond)
// of the parsed string and whether a time zone was
// specified.
//
// Certain metadata formats has limited time representations:
//
// MVHD in MP4 is unix(ish) time lacking time zone information.
//
// Exif has a fixed standard time layout without a time zone.
// Certain tools can write time zone information to Exif date fields,
// but such Exif files are technically invalid.
//
// XMP uses the time format understood by ParseTime, but may
// omit elements from the end of the string, reducing precision.
type Time struct {
	// Actual time value.
	// Its location is always time.Local if HasLoc is false.
	time.Time

	// Prec records the number of valid components of the parsed
	// value between 1 (year) and 7 (subsecond).
	// A Time with Prec == 0 is invalid.
	Prec int

	// HasLoc records whether the parsed value included a time zone.
	HasLoc bool
}

// ParseTime parses a time string based on the RFC 3339 format,
// possibly truncated and with or without a time zone.
func ParseTime(s string) Time {
	tp := timeParser{p: s}

	// TODO: limit length of elements so that
	// 20060102T000000 is not parsed as year 20060102

	year := tp.val(":-")
	month := tp.xval(":-")
	day := tp.xval("tT")

	if tp.prec == 0 {
		return Time{}
	}

	hour := tp.val(":")
	min := tp.val(":")
	sec := tp.val(".")

	nsec, ndenom, ok := tp.rat("")
	if ok {
		for ndenom < 1e9 {
			nsec, ndenom = nsec*10, ndenom*10
		}
	}

	loc := tp.loc()

	return Time{
		Time:   time.Date(year, time.Month(month), day, hour, min, sec, nsec, loc),
		Prec:   tp.prec,
		HasLoc: tp.hasLoc,
	}
}

// In returns t with the location information set to loc.
// If t.HasLoc was false, the time.Time of the result will have
// the same Date() and Clock() as before and its HasLoc set.
//
// In panics if loc is nil.
func (t Time) In(loc *time.Location) Time {
	if t.HasLoc {
		t.Time = t.Time.In(loc)
		return t
	}

	// Zone was not known beforehand.
	// Store old and new offset and adjust time value as needed.
	_, o0 := t.Time.Zone()
	t.Time = t.Time.In(loc)
	_, o1 := t.Time.Zone()
	t.Time = t.Time.Add(time.Duration(o0-o1) * time.Second)
	t.HasLoc = true
	return t
}

var precLayout = []string{
	"2006",
	"2006-01",
	"2006-01-02",
	"2006-01-02T15",
	"2006-01-02T15:04",
	"2006-01-02T15:04:05",
}

// String formats t using the layout understood by ParseTime.
func (t Time) String() string {
	var layout string
	switch {
	case t.Prec <= 0:
		return ""
	case 0 < t.Prec && t.Prec < 7:
		layout = precLayout[t.Prec-1]
	default:
		if t.Time.Nanosecond() == 0 {
			// time.Time.Format would omit the nanosecond
			// part with ".9", therefore ".0" is needed
			// to keep the precision.
			layout = "2006-01-02T15:04:05.0"
		} else {
			layout = "2006-01-02T15:04:05.999999999"
		}
	}
	if t.HasLoc {
		layout += "Z07:00"
	}
	return t.Time.Format(layout)
}

type timeParser struct {
	p string
	r int

	prec   int
	hasLoc bool

	done bool
}

func (p *timeParser) val(sep string) int {
	r, _, _ := p.rat(sep)
	return r
}

func (p *timeParser) xval(sep string) int {
	r, _, ok := p.rat(sep)
	if !ok {
		r = 1
	}
	return r
}

func (p *timeParser) rat(sep string) (num, denom int, ok bool) {
	if p.done {
		return 0, 1, false
	}
	start := p.r
	denom = 1
	for ; p.r < len(p.p); p.r++ {
		c := p.p[p.r]
		if '0' <= c && c <= '9' {
			if denom < 1e9 {
				num = num*10 + int(c-'0')
				denom *= 10
			}
		} else {
			break
		}
	}
	if start == p.r {
		p.done = true
		return 0, 1, false
	}
	p.prec++
	if sep != "" {
		p.sep(sep)
	}
	return num, denom, true
}

func (p *timeParser) sep(chars string) {
	if p.done {
		return
	}
	if p.r < len(p.p) {
		for _, c := range chars {
			if rune(p.p[p.r]) == c {
				p.r++
				return
			}
		}
	}
	p.done = true
}

func (p *timeParser) loc() *time.Location {
	if p.r == len(p.p) {
		return time.Local
	}
	for _, l := range []string{
		"Z07:00",
		"Z0700",
		"Z07:00:00",
		"Z070000",
		"Z07",
	} {
		t, err := time.Parse(l, p.p[p.r:])
		if err == nil {
			p.hasLoc = true
			return t.Location()
		}
	}
	// can't parse location
	return time.Local
}
