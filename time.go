package metadata

import "time"

// Time is like time.Time but records the precision
// (year, month, day, hout, minute, second or subsecond)
// from the parsed value and whether a time zone was
// specified.
type Time struct {
	// Actual time value.
	// Its location is time.Local if ZoneKnown is false.
	time.Time

	// Prec records the number of valid components
	// from the beginning of the RFC3339 format:
	//  0: invalid
	//  1: yyyy
	//  2: yyyy-mm
	//  3: yyyy-mm-dd
	//  4: yyyy-mm-ddThh
	//  5: yyyy-mm-ddThh:mm
	//  6: yyyy-mm-ddThh:mm:ss
	//  7: yyyy-mm-ddThh:mm:ss.ss
	Prec int

	// ZoneKnown is true if source has a specific time zone.
	ZoneKnown bool
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
		Time:      time.Date(year, time.Month(month), day, hour, min, sec, nsec, loc),
		Prec:      tp.prec,
		ZoneKnown: tp.zoneknown,
	}
}

// In returns t with the location information set to loc.
// If t.ZoneKnown was false, the time.Time of the result will have
// the same Date() and Clock() as before and its ZoneKnown set.
//
// In panics if loc is nil.
func (t Time) In(loc *time.Location) Time {
	if t.ZoneKnown {
		t.Time = t.Time.In(loc)
		return t
	}

	// Zone was not known beforehand.
	// Store old and new offset and adjust time value as needed.
	_, o0 := t.Time.Zone()
	t.Time = t.Time.In(loc)
	_, o1 := t.Time.Zone()
	t.Time = t.Time.Add(time.Duration(o0-o1) * time.Second)
	t.ZoneKnown = true
	return t
}

type timeParser struct {
	p string
	r int

	prec      int
	zoneknown bool

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
			p.zoneknown = true
			return t.Location()
		}
	}
	// can't parse location
	return time.Local
}
