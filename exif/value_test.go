package exif

import (
	"encoding/binary"
	"math/big"
	"testing"
	"time"
)

func TestMath128(t *testing.T) {
	// n1 / 7 = q1 rem 6
	const n1 = 36893488147419103234
	const q1 = 5270498306774157604
	testDivmod128(t, n1>>64, n1-(n1>>64)<<64, 7, q1>>64, q1-(q1>>64)<<64, 6)

	// 1e30 / 1e9 = 1e21
	const n2 = 1e30
	const q2 = 1e21
	testDivmod128(t, n2>>64, n2-(n2>>64)<<64, 1e9, q2>>64, q2-(q2>>64)<<64, 0)
}

func testDivmod128(t *testing.T, nhi, nlo, d, wqhi, wqlo, wrem uint64) {
	if wrem >= d {
		t.Fatal("invalid remainder >= divisor")
	}
	gqhi, gqlo, grem := divmod128(nhi, nlo, d)
	if gqhi != wqhi || gqlo != wqlo || grem != wrem {
		t.Errorf("divmod128(%v, %v, %v) is (%v, %v, %v), want (%v, %v, %v)",
			nhi, nlo, d, gqhi, gqlo, grem, wqhi, wqlo, wrem)
	}
}

func TestRational(t *testing.T) {
	// largest possible rational triplet
	const maxuint32 = 1<<32 - 1
	max := Rational{maxuint32, 1, maxuint32, 1, maxuint32, 1}
	hi, lo, ok := max.Sexagesimal(1e6)
	if !ok {
		t.Fatal("conversion failed")
	}
	if hi != 0 {
		t.Error("Rational.Sexagesimal promise invalid")
	}
	want := uint64(maxuint32*3600+maxuint32*60+maxuint32) * 1e6
	if lo != want {
		t.Errorf("Rational.Sexagesimal yields %v, want %v", lo, want)
	}
	testSexagesimal(t, max, 1e6)

	// test strange denominators
	testSexagesimal(t, Rational{maxuint32, 7, maxuint32, 13, maxuint32, 257}, 1e9)

	// times
	testSexagesimalV(t, Rational{24, 1, 0, 1, 0, 1}, uint32(time.Second), uint64(24*60*60*time.Second))
	testSexagesimalV(t, Rational{23, 1, 48, 1, 56, 1}, uint32(time.Second),
		uint64(23*time.Hour+48*time.Minute+56*time.Second))

	// 180 deg with 6 significant digits for second fractions
	testSexagesimalV(t, Rational{180, 1, 0, 1, 0, 1}, 1e6, uint64(180*60*60*1e6))

	// 23° 15' 0.01"
	testSexagesimalV(t, Rational{23, 1, 15, 1, 1, 100}, 1e6, uint64((23*3600+15*60)*1e6+1e4))

	// 85° 5.0012'
	testSexagesimalV(t, Rational{85, 1, 50012, 10000, 0, 1}, 1e6, uint64(85*3600*1e6+50012*60*100))

	// 156.123456°
	testSexagesimalV(t, Rational{156123456, 1e6, 0, 1, 0, 1}, 1e6, uint64(156123456*3600))

	// creation of Sexagesimals
	testMakeSexagesimal(t, uint64(23*time.Hour+48*time.Minute+56*time.Second), uint32(time.Second),
		Rational{23, 1, 48, 1, 56, 1})

	// 23° 15' 0.01"
	testMakeSexagesimal(t, uint64((23*3600+15*60)*1e6+1e4), 1e6,
		Rational{23, 1, 15, 1, 1, 100})

	// resolution overflow with rounding (500/1e9 → 1/1e6)
	testMakeSexagesimal(t, uint64((23*3600+15*60)*1e9+500), 1e9, Rational{23, 1, 15, 1, 1, 1e6})

	// resolution overflow with rounding (499/1e9 → 0)
	testMakeSexagesimal(t, uint64((23*3600+15*60)*1e9+499), 1e9, Rational{23, 1, 15, 1, 0, 1})
}

func testSexagesimal(t *testing.T, r Rational, res uint32) {
	testSexagesimalImpl(t, r, res, 0, false)
}

func testSexagesimalV(t *testing.T, r Rational, res uint32, val uint64) {
	testSexagesimalImpl(t, r, res, val, true)
}

func testSexagesimalImpl(t *testing.T, r Rational, res uint32, val uint64, tval bool) {
	hi, lo, ok := r.Sexagesimal(res)
	if !ok {
		t.Errorf("Rational.Sexagesimal %v invalid", r)
		return
	}

	got := big128(hi, lo)

	want := bigSexagesimal(r, res)

	if got.Cmp(want) != 0 {
		t.Errorf("Rational.Sexagesimal %v yields %v want %v", r, got, want)
	}

	if tval {
		if hi != 0 || lo != val {
			t.Errorf("Rational.Sexagesimal %v yields %v want %v", r, got, val)
		}
	}
}

func testMakeSexagesimal(t *testing.T, val uint64, res uint32, r Rational) {
	// test if Sexagesimal produces the correct result
	xr := Sexagesimal(val, res)
	ok := true
	for i := 0; i < 3; i++ {
		gn, gd := xr[2*i], xr[2*i+1]
		wn, wd := r[2*i], r[2*i+1]
		// a/b == c/d iff a*d == c*b
		v1 := uint64(gn) * uint64(wd)
		v2 := uint64(wn) * uint64(gd)
		ok = ok && v1 == v2
	}
	if !ok {
		t.Errorf("Sexagesimal(%v, %v) yields %v want %v", val, res, xr, r)
	}
}

func big128(hi, lo uint64) *big.Int {
	p := make([]byte, 16)
	binary.BigEndian.PutUint64(p, hi)
	binary.BigEndian.PutUint64(p[8:], lo)

	return new(big.Int).SetBytes(p)
}

// bigSexagesimal is like Rational.Sexagesimal but
// uses math/big internally, and returns a big.Int instead
// of two 64-bit values
func bigSexagesimal(r Rational, res uint32) *big.Int {
	h := big.NewRat(int64(r[0])*3600, int64(r[1]))
	m := big.NewRat(int64(r[2])*60, int64(r[3]))
	s := big.NewRat(int64(r[4]), int64(r[5]))

	z := new(big.Rat)
	z.Add(z, h).Add(z, m).Add(z, s)

	n := new(big.Int)
	n.Mul(z.Num(), big.NewInt(int64(res)))

	rem := new(big.Int)
	n.DivMod(n, z.Denom(), rem)

	if rem.Int64() >= int64(res/2) {
		// round up
		n.Add(n, big.NewInt(1))
	}
	return n
}
