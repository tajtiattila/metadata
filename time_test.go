package metadata

import (
	"testing"
	"time"
)

func TestParseTime(t *testing.T) {
	f := func(s, want string, prec int, zone bool) {
		testParseTime(t, s, want, prec, zone)
	}
	f("0", "0000-01-01T00:00:00", 1, false)
	f("198", "0198-01-01T00:00:00", 1, false)
	f("1984-02", "1984-02-01T00:00:00", 2, false)
	f("1984-02-10", "1984-02-10T00:00:00", 3, false)
	f("1984-02-10T22", "1984-02-10T22:00:00", 4, false)
	f("1984-02-10t22:48", "1984-02-10T22:48:00", 5, false)
	f("1984-02-10t22:48:56", "1984-02-10T22:48:56", 6, false)
	f("1984-02-10t22:48:56.998", "1984-02-10T22:48:56.998", 7, false)
	f("1984-02-10t22:48+0100", "1984-02-10T22:48:00+01:00", 5, true)
	f("1984-02-10t22:48:56+01:00", "1984-02-10T22:48:56+01:00", 6, true)
	f("1984-02-10t22:48:56.998+01:00", "1984-02-10T22:48:56.998+01:00", 7, true)
	f("1984-02-10t22:48:56Z", "1984-02-10T22:48:56Z", 6, true)
	f("1984-02-10t22:48:56.998Z", "1984-02-10T22:48:56.998Z", 7, true)

	testParseTimeZero(t, "")
	testParseTimeZero(t, "foo")
	testParseTimeZero(t, "+02:00")
	testParseTimeZero(t, "Z")
}

func testParseTime(t *testing.T, s, w string, prec int, zone bool) {
	got := ParseTime(s)
	if got.HasLoc != zone {
		t.Errorf("testParseTime %s HasLoc got %v != want %v", s, got.HasLoc, zone)
	}

	var want time.Time
	var err error
	if zone {
		want, err = time.Parse(time.RFC3339, w)
	} else {
		want, err = time.ParseInLocation("2006-01-02T15:04:05.999999", w, time.Local)
	}
	if err != nil {
		t.Error("testParseTime can't parse wanted time", err)
		return
	}
	if want != got.Time {
		t.Errorf("testParseTime %q got %v != want %v", s, got.Time, want)
		t.Error(want.Sub(got.Time))
	}
	if got.Prec != prec {
		t.Errorf("testParseTime %q Prec got %d != want %d", s, got.Prec, prec)
	}
}

func testParseTimeZero(t *testing.T, s string) {
	got := ParseTime(s)
	if !got.Time.IsZero() {
		t.Errorf("ParseTime(%q) got %v, should be zero", s, got.Time)
	}
	if got.HasLoc {
		t.Errorf("ParseTime(%q) got HasLoc=true, want false", s)
	}
	if got.Prec != 0 {
		t.Errorf("ParseTime(%q) got Prec=%d, want 0", s, got.Prec)
	}
}

func TestTimeIn(t *testing.T) {
	l0 := time.Local
	l1, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	l2, err := time.LoadLocation("Europe/Budapest")
	if err != nil {
		t.Fatal(err)
	}

	testTimeIn(t, 2005, time.August, 7, 3, 12, 45, l0, l1)
	testTimeIn(t, 2005, time.August, 7, 3, 12, 45, l1, l2)
	testTimeIn(t, 2005, time.August, 7, 3, 12, 45, l2, l1)
}

func testTimeIn(t *testing.T, y int, month time.Month, d, h, min, s int, l0, l1 *time.Location) {
	src := Time{time.Date(y, month, d, h, min, s, 0, l0), 6, false}
	got := src.In(l1)
	if !got.HasLoc {
		t.Error("Time.In didn't set HasLoc")
	}

	sY, sM, sD := src.Date()
	gY, gM, gD := got.Date()
	if sY != gY || sM != gM || sD != gD {
		t.Errorf("testTimeIn date differ got %v != src %v", got.Time, src.Time)
	}

	sh, sm, ss := src.Clock()
	gh, gm, gs := got.Clock()
	if sh != gh || sm != gm || ss != gs {
		t.Errorf("testTimeIn time differ got %v != src %v", got.Time, src.Time)
	}
}

func TestTimeLoc(t *testing.T) {
	// CET
	testTimeLoc(t, "2018-03-16T18:32:55", "Europe/Berlin", "2018-03-16T18:32:55+01:00")
	testTimeLoc(t, "2018-03-16T18:32:55Z", "Europe/Berlin", "2018-03-16T19:32:55+01:00")
	// CEST
	testTimeLoc(t, "2018-07-16T18:32:55", "Europe/Berlin", "2018-07-16T18:32:55+02:00")
	testTimeLoc(t, "2018-07-16T18:32:55Z", "Europe/Berlin", "2018-07-16T20:32:55+02:00")
}

func testTimeLoc(t *testing.T, src, locname, want string) {
	p := ParseTime(src)

	loc, err := time.LoadLocation(locname)
	if err != nil {
		t.Fatal(err)
	}

	q := p.In(loc)

	got := q.String()
	if got != want {
		t.Errorf("%s in %s is %s, want %s", src, locname, got, want)
	}
}
