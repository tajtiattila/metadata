package exif_test

import (
	"testing"
	"time"

	"github.com/tajtiattila/metadata/exif"
	"github.com/tajtiattila/metadata/exif/exiftag"
)

// TestStdTime ensures time.Time works as expected
// with relation to time.Local.
func TestStdTime(t *testing.T) {
	locnames := []string{
		"America/New_York",
		"Europe/Moscow",
		"Asia/Tokyo",
	}

	for _, locname := range locnames {
		loc, err := time.LoadLocation(locname)
		if err != nil {
			t.Fatalf("can't load location %q", locname)
		}

		const timeFmt = "2006-01-02T15:04:05"
		const src = "2018-03-27T13:24:55"
		st, err := time.ParseInLocation(timeFmt, src, loc)
		if err != nil {
			t.Fatalf("can't parse time %q as %q", src, timeFmt)
		}

		t.Logf("src is %q", st.Format(time.RFC3339))

		x := exif.New(128, 128)
		x.SetTime(exiftag.DateTimeOriginal, exiftag.SubSecTimeOriginal, st)

		dt, _ := x.Time(exiftag.DateTimeOriginal, exiftag.SubSecTimeOriginal)
		if dt.Location() != time.Local {
			t.Error("exif time is not local")
		}

		ds := dt.Format(timeFmt)
		if ds != src {
			t.Errorf("time values differ, want %q got %q", st, ds)
		}
	}
}
