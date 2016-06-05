package mp4

import (
	"encoding/binary"
	"time"
)

type MVHD struct {
	Version      byte
	Flags        [3]byte
	DateCreated  time.Time // value in seconds since beginning 1904 to 2040
	DateModified time.Time // value in seconds since beginning 1904 to 2040

	TimeUnit        uint32 // time unit per second (default = 600)
	DurationInUnits uint64 // time length (in time units)

	Raw []byte // undecoded data after decoded bits above
}

var ErrShortMVHD = formatError("MVHD too short")

func DecodeMVHD(p []byte) (*MVHD, error) {
	if len(p) < 20 {
		return nil, ErrShortMVHD
	}

	m := new(MVHD)
	m.Version = p[0]
	copy(m.Flags[:], p[1:4])
	i := 4

	var getValue func() uint64

	x := binary.BigEndian

	switch p[0] {
	case 0:
		getValue = func() uint64 {
			v := x.Uint32(p[i:])
			i += 4
			return uint64(v)
		}
	case 1:
		if len(p) < 32 {
			return nil, ErrShortMVHD
		}
		getValue = func() uint64 {
			v := x.Uint64(p[i:])
			i += 8
			return v
		}
	default:
		return nil, formatError("Unknown MVHD version %d", p[0])
	}

	// mac UTC date epoch
	epoch := time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC)

	m.DateCreated = epoch.Add(time.Duration(getValue()) * time.Second)
	m.DateModified = epoch.Add(time.Duration(getValue()) * time.Second)
	m.TimeUnit, i = x.Uint32(p[i:]), i+4
	m.DurationInUnits = getValue()

	if i < len(p) {
		m.Raw = make([]byte, len(p)-i)
		copy(m.Raw, p[i:])
	}
	return m, nil
}

// encoded length in bytes
func (m *MVHD) Len() int {
	l := 20 + len(m.Raw)
	if m.Version == 1 {
		l += 12
	}
	return l
}

func (m *MVHD) Duration() time.Duration {
	return time.Duration(m.DurationInUnits) * time.Second / time.Duration(m.TimeUnit)
}

/* MVHD http://xhelmboyx.tripod.com/formats/mp4-layout.txt

   * 8+ bytes movie (presentation) header box
       = long unsigned offset + long ASCII text string 'mvhd'
     -> 1 byte version = 8-bit unsigned value
       - if version is 1 then date and duration values are 8 bytes in length
     -> 3 bytes flags =  24-bit hex flags (current = 0)

     -> 4 bytes created mac UTC date
         = long unsigned value in seconds since beginning 1904 to 2040
     -> 4 bytes modified mac UTC date
         = long unsigned value in seconds since beginning 1904 to 2040
     OR
     -> 8 bytes created mac UTC date
         = 64-bit unsigned value in seconds since beginning 1904
     -> 8 bytes modified mac UTC date
         = 64-bit unsigned value in seconds since beginning 1904

     -> 4 bytes time scale = long unsigned time unit per second (default = 600)

     -> 4 bytes duration = long unsigned time length (in time units)
     OR
     -> 8 bytes duration = 64-bit unsigned time length (in time units)

     -> 4 bytes decimal user playback speed = long fixed point rate (normal = 1.0)
     -> 2 bytes decimal user volume = short fixed point level
         (mute = 0.0 ; normal = 1.0 ; QUICKTIME MAX = 3.0)
     -> 10 bytes reserved = 5 * short values set to zero
     -> 4 bytes decimal window geometry matrix value A
         = long fixed point width scale (normal = 1.0)
     -> 4 bytes decimal window geometry matrix value B
         = long fixed point width rotate (normal = 0.0)
     -> 4 bytes decimal window geometry matrix value U
         = long fixed point width angle (restricted to 0.0)
     -> 4 bytes decimal window geometry matrix value C
         = long fixed point height rotate (normal = 0.0)
     -> 4 bytes decimal window geometry matrix value D
         = long fixed point height scale (normal = 1.0)
     -> 4 bytes decimal window geometry matrix value V
         = long fixed point height angle (restricted to 0.0)
     -> 4 bytes decimal window geometry matrix value X
         = long fixed point positon (left = 0.0)
     -> 4 bytes decimal window geometry matrix value Y
         = long fixed point positon (top = 0.0)
     -> 4 bytes decimal window geometry matrix value W
         = long fixed point divider scale (restricted to 1.0)
     -> 8 bytes QUICKTIME preview
         = long unsigned start time + long unsigned time length (in time units)
     -> 4 bytes QUICKTIME still poster
         = long unsigned frame time (in time units)
     -> 8 bytes QUICKTIME selection time
         = long unsigned start time + long unsigned time length (in time units)
     -> 4 bytes QUICKTIME current time = long unsigned frame time (in time units)
     -> 4 bytes next/new track id = long integer value (single track = 2)
*/
