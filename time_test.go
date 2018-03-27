package metadata_test

import (
	"testing"
	"time"

	"github.com/tajtiattila/metadata"
)

func TestFixLocalTime(t *testing.T) {
	l0 := time.Local
	l1, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	l2, err := time.LoadLocation("Europe/Budapest")
	if err != nil {
		t.Fatal(err)
	}

	testFixLocalTime(t, 2005, time.August, 7, 3, 12, 45, l0, l1)
	testFixLocalTime(t, 2005, time.August, 7, 3, 12, 45, l1, l2)
	testFixLocalTime(t, 2005, time.August, 7, 3, 12, 45, l2, l1)
}

func testFixLocalTime(t *testing.T, y int, month time.Month, d, h, min, s int, l0, l1 *time.Location) {
	localSave := time.Local
	time.Local = l0
	defer func() {
		time.Local = localSave
	}()

	src := time.Date(y, month, d, h, min, s, 0, time.Local)
	got := metadata.FixLocalTime(src, l1)

	if got.Location() != l1 {
		t.Error("FixLocalTime didn't set location")
	}
	sameTime(t, "testTimeIn", got, src)
}

func sameTime(t *testing.T, fn string, got, want time.Time) {
	sY, sM, sD := want.Date()
	gY, gM, gD := got.Date()
	if sY != gY || sM != gM || sD != gD {
		t.Errorf("%s date differ got %v, want %v", fn, got.Format(time.RFC3339), want.Format(time.RFC3339))
	}

	sh, sm, ss := want.Clock()
	gh, gm, gs := got.Clock()
	if sh != gh || sm != gm || ss != gs {
		t.Errorf("%s time differ got %v, want %v", fn, got.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}
