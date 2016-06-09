package exif

import (
	"encoding/binary"
	"errors"
	"sort"
)

const (
	// sub-IFD names
	ifd0exifSub    = 0x8769
	ifd0gpsSub     = 0x8825
	ifd0interopSub = 0xA005

	// other data
	ifd1thumbOffset = 0x201
	ifd1thumbLength = 0x202
)

var (
	ErrCorruptHeader = errors.New("exif: corrupt header")
	ErrCorruptDir    = errors.New("exif: corrupt IFD")
	ErrCorruptTag    = errors.New("exif: corrupt IFD tag")
	ErrDuplicateSub  = errors.New("exif: duplicate sub-IFD entry")

	// ErrEmpty is returned when x.Encode is used with no exif data to encode.
	ErrEmpty = errors.New("exif: nothing to encode")

	// ErrTooLong is returned if the serialized exif is too long to be written in an Exif file.
	ErrTooLong = errors.New("exif: encoded length too long")
)

// DecodeBytes decodes the raw Exif data from p.
func DecodeBytes(p []byte) (*Exif, error) {
	if len(p) < 4 {
		// header too short
		return nil, ErrCorruptHeader
	}

	var bo binary.ByteOrder
	switch {
	case p[0] == 'M' && p[1] == 'M':
		bo = binary.BigEndian
	case p[0] == 'I' && p[1] == 'I':
		bo = binary.LittleEndian
	default:
		// invalid byte order
		return nil, ErrCorruptHeader
	}

	if bo.Uint16(p[2:]) != 42 {
		// invalid IFD tag
		return nil, ErrCorruptHeader
	}

	// location of IFD0 offset
	offset := 4

	var d []Dir
	var err error
	for {
		if len(p) < offset+4 {
			// offset points outside exif
			return nil, ErrCorruptDir
		}
		ptr := int(bo.Uint32(p[offset:]))
		if ptr == 0 {
			break
		}
		if ptr < 0 || len(p) < ptr {
			// corrupt IFD offset in header
			return nil, ErrCorruptDir
		}

		var dir Dir
		dir, offset, err = decodeDir(bo, p, ptr)
		if err != nil {
			return nil, err
		}
		d = append(d, dir)
	}

	// populate sub-IFDs
	var ifd0, ifd1 Dir
	if len(d) > 0 {
		ifd0 = d[0]
		if len(d) > 1 {
			ifd1 = d[1]
		}
	}
	x := &Exif{ByteOrder: bo, IFD0: ifd0, IFD1: ifd1}
	for _, t := range ifd0 {
		var psub *Dir
		switch t.Tag {
		case ifd0exifSub:
			psub = &x.Exif
		case ifd0gpsSub:
			psub = &x.GPS
		case ifd0interopSub:
			psub = &x.Interop
		default:
			continue
		}
		if *psub != nil {
			// sub-IFD already loaded
			return nil, ErrDuplicateSub
		}
		if t.Type != TypeLong {
			// pointer must be a long
			return nil, ErrCorruptTag
		}
		ptr := int(bo.Uint32(t.Value))
		if ptr < 0 || len(p) < ptr {
			// invalid pointer
			return nil, ErrCorruptTag
		}
		subdir, _, err := decodeDir(bo, p, ptr)
		if err != nil {
			return nil, err
		}
		*psub = subdir
	}

	// Preserve raw thumb data
	tofs, tlen, ok := getOffsetLen(bo, ifd1, ifd1thumbOffset, ifd1thumbLength)
	if ok && 0 <= tofs && tofs+tlen <= len(p) {
		x.Thumb = make([]byte, tlen)
		copy(x.Thumb, p[tofs:tofs+tlen])
	}

	return x, nil
}

// EncodeBytes encodes Exif data as a byte slice.
// It returns an error only if IFD0 is empty, the byte order is not set
// or the encoded length is too long for Exif.
//
// To store the Exif within an image, use Copy instead.
func (x *Exif) EncodeBytes() ([]byte, error) {
	// prepare sub-IFDs
	subifd := []struct {
		idx int // within IFD0
		tag uint16
		dir Dir
	}{
		{-1, ifd0exifSub, x.Exif},
		{-1, ifd0gpsSub, x.GPS},
		{-1, ifd0interopSub, x.Interop},
	}

	// filter/set IFD0 to have the needed subifds
	var ifd0 Dir

Outer:
	for i, t := range x.IFD0 {
		for j := range subifd {
			sub := &subifd[j]
			if t.Tag == sub.tag {
				if len(sub.dir) == 0 {
					// skip empty sub-IFD
					continue Outer
				}
				sub.idx = i
			}
		}
		ifd0 = append(ifd0, t)
	}

	// add missing pointers
	for j := range subifd {
		sub := &subifd[j]
		if len(sub.dir) != 0 && sub.idx == -1 {
			sub.idx = len(ifd0)
			ifd0 = append(ifd0, Entry{
				Tag:   sub.tag,
				Type:  TypeLong,
				Count: 1,
				Value: make([]byte, 4),
			})
		}
	}

	bo := x.ByteOrder

	// perpare thumb
	ifd1 := x.IFD1
	thumb := x.Thumb

	// check if thumb has data, and we can set its offset and length in ifd1
	if len(thumb) == 0 ||
		!putOffsetLen(bo, ifd1, ifd1thumbOffset, ifd1thumbLength, 0, 0) {

		// can't write thumb
		// TODO add ifd1thumbOffset/ifd1thumbLength if x.Thumb != nil?

		// drop thumb
		ifd1 = nil
		thumb = nil
	}

	// calc final dirs
	dirs := []Dir{ifd0}
	if len(ifd1) != 0 {
		dirs = append(dirs, ifd1)
	}

	if len(dirs) == 0 {
		return nil, ErrEmpty
	}

	switch bo {
	case binary.BigEndian:
	case binary.LittleEndian:
		// pass
	default:
		return nil, ErrCorruptHeader
	}

	// calculate initial offset for sub-IFDs
	suboffset := 8 // endianness, magic, 1st IFD pointer
	for _, d := range dirs {
		suboffset += d.encodedLen(false)
	}

	// set sub-IFD offsets within IFD0
	for _, sub := range subifd {
		if sub.idx != -1 {
			t := ifd0[sub.idx]
			bo.PutUint32(t.Value, uint32(suboffset))
			suboffset += sub.dir.encodedLen(true)
		}
	}

	// set thumbnail offset
	if len(thumb) != 0 {
		ok := putOffsetLen(bo, ifd1, ifd1thumbOffset, ifd1thumbLength, suboffset, len(thumb))
		if !ok {
			panic("impossible")
		}
	}
	suboffset += len(thumb)

	p := make([]byte, 8+suboffset)

	// write header
	switch bo {
	case binary.BigEndian:
		p[0] = 'M'
		p[1] = 'M'
	case binary.LittleEndian:
		p[0] = 'I'
		p[1] = 'I'
	}
	bo.PutUint16(p[2:], 42)
	offset := 8
	bo.PutUint32(p[4:], uint32(offset))

	// TODO write IFD0 and its sub-IFDs before IFD1?
	// It doesn't seem necessary, but that is how the order is presented in the Exif 2.2 spec

	// write IFDs
	for i, d := range dirs {
		offset = d.encode(bo, p, offset, false, i+1 != len(dirs))
	}

	// write sub-IFDs
	for _, sub := range subifd {
		if sub.idx != -1 {
			offset = sub.dir.encode(bo, p, offset, true, false)
		}
	}

	// write thumb
	copy(p[offset:offset+len(thumb)], thumb)

	if len(p) > 65533 {
		return p, ErrTooLong
	}

	return p, nil
}

// Dir represents an Image File Directory (IFD) within Exif.
// It is a directory of raw tagged fields, also named entries.
type Dir []Entry

func decodeDir(bo binary.ByteOrder, p []byte, offset int) (Dir, int, error) {
	if len(p) < offset+2 {
		return nil, 0, ErrCorruptDir
	}
	ntags := int(bo.Uint16(p[offset:]))
	offset += 2

	var tags []Entry
	for i := 0; i < ntags; i++ {
		// decode entry header
		if len(p) < int(offset+12) {
			return nil, 0, ErrCorruptTag
		}
		tag := bo.Uint16(p[offset:])
		typ := bo.Uint16(p[offset+2:])
		count := bo.Uint32(p[offset+4:])
		valuebits := p[offset+8 : offset+12]
		offset += 12

		// return early for invalid count
		nbytes := typeSize(typ, count)
		if nbytes == 0 {
			return nil, 0, ErrCorruptDir
		}

		// If value doesn't fit in tag header,
		// then it is an offset from the start
		// of the tiff header (EXIF 2.2 §4.6.2).
		if nbytes > 4 {
			n := int(nbytes)
			valueoffset := int(bo.Uint32(valuebits))
			if valueoffset < 0 || len(p) < valueoffset+n {
				return nil, 0, ErrCorruptDir
			}
			valuebits = p[valueoffset : valueoffset+n]
		} else {
			valuebits = valuebits[:nbytes]
		}

		// make a copy of the value for the tag
		value := make([]byte, len(valuebits))
		copy(value, valuebits)

		tags = append(tags, Entry{
			Tag:   tag,
			Type:  typ,
			Count: count,
			Value: value,
		})
	}

	// Tags should appear sorted according to TIFF spec,
	// and it will help in searching as well.
	d := Dir(tags)
	d.Sort()

	return d, offset, nil
}

func (d Dir) encodedLen(subIfd bool) int {
	// tags
	n := 2 + len(d)*12

	if !subIfd {
		// next IFD pointer
		n += 4
	}

	for _, t := range d {
		if len(t.Value) > 4 {
			// tag data
			n += len(t.Value)
		}
	}

	return n
}

func (d Dir) encode(bo binary.ByteOrder, p []byte, offset int, subIfd, hasNext bool) int {
	// offset for data outside tag header
	dataoffset := offset + 2 + len(d)*12
	if !subIfd {
		dataoffset += 4
	}

	bo.PutUint16(p[offset:], uint16(len(d)))
	offset += 2

	for _, t := range d {
		bo.PutUint16(p[offset:], t.Tag)
		bo.PutUint16(p[offset+2:], t.Type)
		bo.PutUint32(p[offset+4:], t.Count)
		if len(t.Value) <= 4 {
			copy(p[offset+8:], t.Value)
		} else {
			bo.PutUint32(p[offset+8:], uint32(dataoffset))
			copy(p[dataoffset:], t.Value)
			dataoffset += len(t.Value)
		}
		offset += 12
	}

	if !subIfd {
		// write next IFD pointer (or leave as zero)
		if hasNext {
			bo.PutUint32(p[offset:], uint32(dataoffset))
		}
	}

	return dataoffset
}

// Sort sorts entries according to tag values, as needed by Tag() and Index().
//
// Tags should appear sorted according to TIFF spec, therefore
// the functions of this package keep Dirs are always sorted.
func (d Dir) Sort() {
	sort.Sort(dirSort(d))
}

// Tag returns a pointer to the Entry with tag t, or nil if t does not exist.
func (d Dir) Tag(t uint16) *Entry {
	i := d.Index(t)
	if i != -1 {
		return &d[i]
	}
	return nil
}

// Index returns the index of tag t, or -1 if t does not exist in d.
func (d Dir) Index(t uint16) int {
	i := sort.Search(len(d), func(i int) bool {
		return t <= d[i].Tag
	})
	if i == len(d) || d[i].Tag != t {
		return -1
	}
	return i
}

// EnsureTag returns a pointer to the Entry with tag t.
//
// An empty Entry with no Type or Count is created if t does not exist in d.
func (d *Dir) EnsureTag(t uint16) *Entry {
	i := sort.Search(len(*d), func(i int) bool {
		return t <= (*d)[i].Tag
	})
	switch {
	case i == len(*d):
		*d = append(*d, Entry{Tag: t})
	case (*d)[i].Tag != t:
		*d = append(*d, Entry{})
		copy((*d)[i+1:], (*d)[i:])
		(*d)[i] = Entry{Tag: t}
	}
	return &(*d)[i]
}

// Remove removes t from d.
func (d *Dir) Remove(t uint16) {
	i := d.Index(t)
	if i == -1 {
		return
	}

	copy((*d)[i:], (*d)[i+1:])
	*d = (*d)[:len(*d)-1]
}

type dirSort []Entry

func (s dirSort) Len() int           { return len(s) }
func (s dirSort) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s dirSort) Less(i, j int) bool { return s[i].Tag < s[j].Tag }

func getOffset(bo binary.ByteOrder, d Dir, ofst uint16) (offset int, ok bool) {
	return fieldOfs(bo, d.Tag(ofst))
}

func getOffsetLen(bo binary.ByteOrder, d Dir, ofst, lent uint16) (offset, length int, ok bool) {
	offset, ok = fieldOfs(bo, d.Tag(ofst))
	if !ok {
		return
	}
	length, ok = fieldOfs(bo, d.Tag(lent))
	return
}

func putOffsetLen(bo binary.ByteOrder, d Dir, ofst, lent uint16, offset, length int) (ok bool) {
	ok = putFieldOfs(bo, d.Tag(ofst), offset)
	ok = ok && putFieldOfs(bo, d.Tag(lent), length)
	return ok
}
