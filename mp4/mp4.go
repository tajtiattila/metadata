package mp4

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
)

type File struct {
	Box []Box

	Size int64

	// MDatOffset is the offset to the "mdat" box.
	// It is used to adjust offsets in "stco" and "co64".
	MDatOffset int64
}

type Box struct {
	Type string // 4-byte cc4
	Size int64

	Raw []byte // raw content, if loaded

	Child []Box // child boxes
}

func Parse(r io.Reader) (*File, error) {
	p := parser{r: r, f: new(File)}
	if err := p.Parse(); err != nil {
		return nil, err
	}
	return p.f, nil
}

type parser struct {
	r io.Reader
	f *File

	off int64
}

const maxParseSize = 1 << 20

func (p *parser) Parse() error {
	for {
		off := p.off
		cc4, size, err := ReadAtomHeader(p.r)
		if err != nil {
			return err
		}
		if p.f.Box == nil {
			if cc4 != "ftyp" {
				return FormatError("ftyp missing")
			}
			if size > 1<<10 {
				return FormatError("ftyp too long")
			}
		}
		if cc4 == "mvhd" {
			if p.f.MDatOffset != 0 {
				return FormatError("multiple mdat")
			}
			p.f.MDatOffset = off
		}
		if size == -1 {
			return p.finish(cc4)
		}
		b := Box{Type: cc4, Size: size}
		if wantBox(cc4) {
			if size > maxParseSize {
				return FormatError("%s too long", cc4)
			}
			b.Raw = make([]byte, int(size))
			if _, err := io.ReadFull(p.r, b.Raw); err != nil {
				return err
			}
			off += int64(len(b.Raw))
		} else {
			if err := p.skip(size); err != nil {
				return err
			}
		}
		p.f.Box = append(p.f.Box, b)
	}
	return nil
}

func (p *parser) finish(cc4 string) error {
	b := Box{Type: cc4, Size: -1}
	if !wantBox(cc4) {
		// unneeded box goes till EOF
		p.f.Box = append(p.f.Box, b)
		return nil
	}
	var err error
	b.Raw, err = ioutil.ReadAll(io.LimitReader(p.r, maxParseSize+1))
	if len(b.Raw) > maxParseSize {
		return FormatError("%s too long", cc4)
	}
	if err != nil {
		return err
	}
	b.Size = int64(len(b.Raw))
	p.f.Box = append(p.f.Box, b)
	return nil
}

func (p *parser) skip(n int64) error {
	var err error
	if s, ok := p.r.(io.Seeker); ok {
		_, err = s.Seek(n, 1)
	} else {
		_, err = io.Copy(ioutil.Discard, io.LimitReader(p.r, n))
	}
	if err == nil {
		p.off += n
	}
	return err
}

func wantBox(cc4 string) bool {
	switch cc4 {
	case "ftyp":
		return true
	case "moov":
		return true
	case "uuid":
		return true
	}
	return false
}

// read next non-empty atom header
func ReadAtomHeader(r io.Reader) (cc4 string, size int64, err error) {
	p := make([]byte, 8)
	for {
		if _, err = io.ReadFull(r, p); err != nil {
			return "", 0, err
		}
		size = int64(binary.BigEndian.Uint32(p))
		cc4 = string(p[4:])

		if size < 8 {
			switch size {
			case 0:
				// box goes until limit (end of enclosing box or EOF)
				size = -1
			case 1:
				// 64-bit box header
				if _, err = io.ReadFull(r, p); err != nil {
					return cc4, 0, err
				}
				size = int64(binary.BigEndian.Uint64(p))
				if size < 16 {
					return cc4, size, fmt.Errorf("invalid extended box size %d", size)
				}
				size -= 8
			default:
				return cc4, size, fmt.Errorf("invalid box size %d", size)
			}
		} else {
			size -= 8
		}

		return cc4, size, nil
	}
	panic("unreachable")
}

func FormatError(f string, args ...interface{}) error {
	return fmt.Errorf(f, args...)
}
