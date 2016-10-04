package mp4

import (
	"encoding/binary"
	"time"
)

var mp4bo = binary.BigEndian

type boxParse struct {
	data []byte
	big  bool // if true, date and duration values are 8 bytes, otherwise 4 bytes
	i    int  // read position

	short   bool
	scratch [8]byte
}

func newBoxParse(p []byte) *boxParse {
	return &boxParse{data: p}
}

func (p *boxParse) versionFlags() (ver byte, flags [3]byte, err error) {
	b := p.next(4)
	ver = b[0]
	copy(flags[:], b[1:])

	if ver > 1 {
		return ver, flags, formatError("Unknown MVHD version %d", ver)
	}

	p.big = ver == 1

	return ver, flags, nil
}

func (p *boxParse) next(n int) []byte {
	i := p.i
	p.i += n
	if p.i <= len(p.data) {
		return p.data[i:p.i]
	}
	p.short = true
	return p.scratch[:n]
}

func (p *boxParse) Skip(n int) {
	p.i += n
	if p.i > len(p.data) {
		p.short = true
	}
}

func (p *boxParse) Short() bool {
	return p.short
}

func (p *boxParse) Rest() []byte {
	if p.short {
		return nil
	}
	return p.data[p.i:]
}

func (p *boxParse) Uint32() uint32 {
	return mp4bo.Uint32(p.next(4))
}

var macUTCepoch = time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC)

func (p *boxParse) Date() time.Time {
	return macUTCepoch.Add(time.Duration(p.UintVar()) * time.Second)
}

func (p *boxParse) UintVar() uint64 {
	if p.big {
		return mp4bo.Uint64(p.next(8))
	} else {
		return uint64(mp4bo.Uint32(p.next(4)))
	}
}
