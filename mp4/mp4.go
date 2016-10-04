package mp4

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
)

type File struct {
	Box

	Header *MVHD // movid header
}

// Parse parses an MP4 file from r.
// If r is a io.ReadSeeker then it is used
// to seek forward within r when necessary.
func Parse(r io.Reader) (*File, error) {
	p := parser{
		r: r,
		f: &File{
			Box: Box{Type: "MP4", Size: -1},
		},
	}
	if err := p.Parse(); err != nil {
		return nil, err
	}

	// parse moov box
	f := p.f
	moov := f.Box.Find("moov")
	if moov == nil {
		return nil, formatError("moov missing")
	}

	if err := moov.unpackChildren(); err != nil {
		return nil, err
	}

	// decode moov.mvhd box
	mvhd := moov.Find("mvhd")
	if mvhd == nil {
		return nil, formatError("mvhd missing")
	}

	h, err := DecodeMVHD(mvhd.Raw)
	if err != nil {
		return nil, err
	}
	f.Header = h

	// calc file size
	for _, b := range f.Child {
		f.Size += b.Size
	}

	return f, nil
}

// AddUuid inserts the an uuid box into f.
// The first 16 bits of data is the 16-byte UUID.
func (f *File) AddUuid(data []byte) {
	if len(data) < 16 {
		panic("len(data) < 16")
	}

	newBox := Box{
		Offset: -1,
		Size:   boxSize(len(data)),
		Type:   "uuid",
		Raw:    data,
	}

	existing := -1
	for i, b := range f.Child {
		if b.Type != "uuid" || len(b.Raw) < 16 {
			continue
		}
		if bytes.Equal(data[:16], b.Raw[:16]) {
			existing = i
			break
		}
	}

	if existing != -1 {
		if f.replace(existing, newBox) {
			// successfully replaced old box in file
			return
		}
		// delete old box
		f.Child[existing] = Box{
			Offset: -1,
			Size:   f.Child[existing].Size,
			Type:   "free",
		}
	}

	if room, _ := f.findFreeSpace(newBox.Size); room != -1 {
		// place box in free space
		ok := f.replace(room, newBox)
		if !ok {
			panic("impossible")
		}
	} else {
		// no space, append box at end
		f.Child = append(f.Child, newBox)
	}
}

func (f *File) replace(idx int, newBox Box) bool {
	oldBox := f.Child[idx]

	// find free boxes after oldBox until there is enough room
	space := oldBox.Size
	next := idx + 1
	for next < len(f.Child) &&
		newBox.Size != space &&
		!boxSizePossible(newBox.Size, space) {
		n := f.Child[next]
		if n.Type != "free" {
			break
		}
		space += n.Size
		next++
	}

	if !boxSizePossible(newBox.Size, space) {
		// no space
		return false
	}

	f.Child[idx] = newBox
	idx++

	// remove consumed free boxes
	ndrop := next - idx
	if newBox.Size < space {
		// want free box after idx
		ndrop--
	}

	switch {
	case ndrop < 0:
		// ndrop == -1: add one
		f.Child = append(f.Child, Box{})
		copy(f.Child[idx+1:], f.Child[next:])
	case ndrop > 0:
		copy(f.Child[idx:], f.Child[idx+ndrop:])
		f.Child = f.Child[:len(f.Child)-next+idx]
	}

	if newBox.Size == space {
		return true
	}

	f.Child[idx] = Box{
		Offset: -1,
		Size:   space - newBox.Size,
		Type:   "free",
	}
	return true
}

func (f *File) findFreeSpace(boxSize int64) (idx, nbox int) {
	var space int64
	for i, b := range f.Child {
		if b.Type == "free" {
			space += b.Size
			if boxSizePossible(boxSize, space) {
				return idx, i - idx + 1
			}
		} else {
			space, idx = 0, i+1
		}
	}
	return -1, 0
}

type Box struct {
	Offset, Size int64 // size and offset of the box in the original file

	Ext bool // false: 8-byte header, true: 16-byte header

	Type string // 4-byte cc4

	Raw []byte // raw content, if loaded

	Child []Box // child boxes
}

func (b *Box) HeaderSize() int64 {
	if b.Ext {
		return 16
	} else {
		return 8
	}
}

// ContentSize returns the payload length, or -1 if the
// box goes to EOF (or the end of its parent)
func (b *Box) ContentSize() int64 {
	if b.Size == 0 {
		return -1
	}
	return b.Size - b.HeaderSize()
}

func (b *Box) Find(typ ...string) *Box {
	if len(typ) == 0 {
		return b
	}
	for i := range b.Child {
		child := &b.Child[i]
		if child.Type == typ[0] {
			return child.Find(typ[1:]...)
		}
	}
	return nil
}

var parentBoxes = setOf("moov", "trak", "mdia", "minf", "stbl")

func (b *Box) unpackChildren() error {
	if _, ok := parentBoxes[b.Type]; !ok {
		return nil
	}

	for off := 0; off < len(b.Raw); {
		if len(b.Raw[off:]) < 8 {
			return formatError("%s unpack", b.Type)
		}
		c := Box{
			Offset: b.Offset + int64(off),
			Size:   int64(binary.BigEndian.Uint32(b.Raw[off:])),
			Type:   string(b.Raw[off+4 : off+8]),
		}

		if c.Size == 1 {
			c.Ext = true
			off += 8
			if len(b.Raw[off:]) < 8 {
				return formatError("%s/%s unpack EOF", b.Type, c.Type)
			}
			// 64-bit box header
			c.Size = int64(binary.BigEndian.Uint64(b.Raw[off:]))
		}
		if c.Size == 0 {
			c.Size = int64(len(b.Raw)-off) + b.HeaderSize()
		}
		if c.Size < c.HeaderSize() {
			return fmt.Errorf("box size %d < %d invalid", c.Size, c.HeaderSize())
		}

		off += 8
		datalen := c.ContentSize()
		if int64(len(b.Raw)) < int64(off)+datalen {
			return formatError("%s/%s unpack EOF", b.Type, c.Type, c.Size)
		}
		c.Raw = b.Raw[off : off+int(datalen)]
		b.Child = append(b.Child, c)
		off += int(datalen)
	}

	for i := range b.Child {
		c := &b.Child[i]
		if err := c.unpackChildren(); err != nil {
			return err
		}
	}
	return nil
}

func (b *Box) packChildren() {
	b.Size = b.packedSize()
	p := make([]byte, int(b.Size))
	off := packBox(b, p, 0)
	if int64(off) != b.Size {
		panic("consistency")
	}
	b.Raw = p
}

func (b *Box) packedSize() int64 {
	if b.Child == nil {
		return boxSize(len(b.Raw))
	}

	var n int64
	for _, c := range b.Child {
		n += c.packedSize()
	}
	return boxSize(int(n))
}

func packBox(b *Box, p []byte, off int) (noff int) {
	// write cc4
	copy(p[off+4:off+8], b.Type)

	// write size
	size := b.packedSize()
	if size < 1<<32 {
		binary.BigEndian.PutUint32(p[off:], 1)
		binary.BigEndian.PutUint64(p[off+8:], uint64(size))
		off += 16
	} else {
		binary.BigEndian.PutUint32(p[off:], uint32(size))
		off += 8
	}

	// write raw content if no children
	if b.Child == nil {
		copy(p[off:], b.Raw)
		return off + len(b.Raw)
	}

	// write children
	for i := range b.Child {
		off = packBox(&b.Child[i], p, off)
	}
	return off
}

func setOf(v ...string) map[string]struct{} {
	m := make(map[string]struct{})
	for _, k := range v {
		m[k] = struct{}{}
	}
	return m
}

type parser struct {
	r io.Reader
	f *File

	off int64 // offset in r

	tmp []byte // scratch buffer
}

const maxParseSize = 1 << 20

func (p *parser) Parse() error {
	for {
		b, err := p.readAtomHeader()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if p.f.Child == nil {
			if b.Type != "ftyp" {
				return formatError("ftyp missing")
			}
			if b.Size > 1<<10 {
				return formatError("ftyp too long")
			}
		}
		if b.Size == 0 {
			return p.finish(b)
		}
		contentSize := b.ContentSize()
		if wantBox(b.Type) {
			if contentSize > maxParseSize {
				return formatError("%s too long", b.Type)
			}
			b.Raw = make([]byte, int(contentSize))
			if _, err := io.ReadFull(p.r, b.Raw); err != nil {
				return err
			}
			p.off += int64(len(b.Raw))
		} else {
			if err := p.skip(contentSize); err != nil {
				return err
			}
		}
		p.f.Child = append(p.f.Child, b)
	}
	return nil
}

func (p *parser) finish(b Box) error {
	if !wantBox(b.Type) {
		// unneeded box goes till EOF
		p.f.Child = append(p.f.Child, b)
		return nil
	}
	var err error
	b.Raw, err = ioutil.ReadAll(io.LimitReader(p.r, maxParseSize+1))
	if len(b.Raw) > maxParseSize {
		return formatError("%s too long", b.Type)
	}
	if err != nil {
		return err
	}
	b.Size = int64(len(b.Raw)) + b.HeaderSize()
	p.f.Child = append(p.f.Child, b)
	return nil
}

func (p *parser) skip(n int64) error {
	var err error
	if s, ok := p.r.(io.Seeker); ok {
		_, err = s.Seek(n, 1)
	} else {
		if p.tmp == nil {
			p.tmp = make([]byte, 32*1024)
		}
		_, err = io.CopyBuffer(ioutil.Discard, io.LimitReader(p.r, n), p.tmp)
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

// read next atom header
func (p *parser) readAtomHeader() (b Box, err error) {
	x := make([]byte, 8)
	var n int
	b.Offset = p.off
	n, err = io.ReadFull(p.r, x)
	p.off += int64(n)
	if err != nil {
		return Box{}, err
	}
	b.Size = int64(binary.BigEndian.Uint32(x))
	b.Type = string(x[4:])

	if b.Size == 1 {
		b.Ext = true
		// 64-bit box header
		n, err = io.ReadFull(p.r, x)
		p.off += int64(n)
		if err != nil {
			return Box{}, err
		}
		b.Size = int64(binary.BigEndian.Uint64(x))
	}
	if b.Size != 0 && b.Size < b.HeaderSize() {
		return Box{}, fmt.Errorf("box size %d < %d invalid", b.Size, b.HeaderSize())
	}

	return b, nil
}

// headerSize returns the header size needed for contentlen.
func headerSize(contentlen int) int64 {
	if contentlen+8 >= 1<<32 {
		return 16
	} else {
		return 8
	}
}

func boxSize(contentlen int) int64 {
	return headerSize(contentlen) + int64(contentlen)
}

func boxSizePossible(size, space int64) bool {
	if size == space {
		return true
	}
	return space-size >= 8
}

func formatError(f string, args ...interface{}) error {
	return fmt.Errorf(f, args...)
}
