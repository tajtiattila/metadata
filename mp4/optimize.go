package mp4

import (
	"encoding/binary"
	"sort"
)

func (f *File) Optimize() error {
	sort.Stable(topLevelBoxSort(f.Child))

	moov := f.Box.Find("moov")
	if moov == nil {
		// no offsets to adjust
		return nil
	}

	// collect old mdat offsets
	var oo []int64
	for _, b := range f.Child {
		if b.Type == "mdat" {
			oo = append(oo, b.Offset)
		}
	}

	// calc size excluding moov
	var lenxmoov int64
	for _, b := range f.Child {
		if b.Type != "moov" {
			lenxmoov += b.Size
		}
	}

	// calc final moov size
	baselen, noffs := analyseMoov(moov)
	use64bit := false
	if s32 := lenxmoov + baselen + 4*int64(noffs); s32 >= 1<<32 {
		use64bit = true
		moov.Size = lenxmoov + baselen + 8*int64(noffs)
	} else {
		moov.Size = s32
	}

	// calc new mdat offsets
	var no []int64
	off := int64(0)
	for _, b := range f.Child {
		if b.Type == "mdat" {
			no = append(no, off)
		}
		off += b.Size
	}

	shiftMoovOffsets(moov, oo, no, use64bit)

	ms := moov.Size
	moov.packChildren()
	if moov.Size != ms {
		panic("consistency")
	}

	return nil
}

type topLevelBoxSort []Box

func (s topLevelBoxSort) Len() int      { return len(s) }
func (s topLevelBoxSort) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s topLevelBoxSort) Less(i, j int) bool {
	return boxIdx(s[i].Type) < boxIdx(s[j].Type)
}

func boxIdx(cc4 string) int {
	switch cc4 {
	case "ftyp":
		return 0
	case "moov":
		return 1
	case "uuid":
		return 2
	default:
		return 3
	case "mdat":
		return 4
	}
}

func analyseMoov(moov *Box) (baselen int64, noffsets int) {
	for _, c := range moov.Child {
		baselen += c.Size
		if c.Type == "trak" {
			stbl := c.Find("mdia", "minf", "stbl")
			if stbl == nil {
				continue
			}
			stco, co64 := stbl.Find("stco"), stbl.Find("co64")
			if n, _ := checkOffsetBlock(stco); n != 0 {
				noffsets += n
				if co64 != nil {
					// has both co64
					baselen -= co64.Size
				}
			} else {
				if n, _ := checkOffsetBlock(co64); n != 0 {
					noffsets += n
				}
			}
		}
	}
	return baselen, noffsets
}

func shiftMoovOffsets(moov *Box, oo, no []int64, use64bit bool) {
	if len(oo) == 0 {
		return
	}

	var offsets []int64
	for _, c := range moov.Child {
		if c.Type == "trak" {
			stbl := c.Find("mdia", "minf", "stbl")
			if stbl == nil {
				continue
			}

			// read old offsets
			stco, co64 := stbl.Find("stco"), stbl.Find("co64")
			n, xf := checkOffsetBlock(stco)
			if n == 0 {
				n, xf = checkOffsetBlock(co64)
			}
			if n == 0 {
				continue
			}
			offsets = offsets[:0]
			for i := 0; i < n; i++ {
				offsets = append(offsets, xf(i))
			}

			// shift offsets
			for i, off := range offsets {
				idx := sort.Search(len(oo), func(i int) bool {
					return oo[i] > off
				})
				if idx == 0 {
					// offset before first mdat
					continue
				}
				idx--
				offsets[i] = off - oo[idx] + no[idx]
			}

			// create box with new offsets
			var cc4 string
			var p []byte
			if use64bit {
				cc4 = "co64"
				p = make([]byte, 8+8*len(offsets))
				binary.BigEndian.PutUint32(p[4:], uint32(len(offsets)))
				for i, off := range offsets {
					binary.BigEndian.PutUint64(p[8+i*8:], uint64(off))
				}
			} else {
				cc4 = "stco"
				p = make([]byte, 4+4*len(offsets))
				binary.BigEndian.PutUint32(p, uint32(len(offsets)))
				for i, off := range offsets {
					binary.BigEndian.PutUint32(p[4+i*4:], uint32(off))
				}
			}
			newBox := Box{
				Offset: -1,
				Size:   boxSize(len(p)),
				Type:   cc4,
				Raw:    p,
			}

			// delete old stco and co64 boxes, add new box
			xc := stbl.Child
			stbl.Child = make([]Box, 0, len(xc))
			for _, c := range xc {
				if c.Type == "stco" || c.Type == "co64" {
					continue
				}
				stbl.Child = append(stbl.Child, c)
			}
			stbl.Child = append(stbl.Child, newBox)
		}
	}
}

func checkOffsetBlock(b *Box) (noffsets int, extract func(i int) int64) {
	if b == nil {
		return 0, nil
	}
	switch b.Type {
	case "stco":
		if len(b.Raw) < 4 {
			return 0, nil
		}
		n := int(binary.BigEndian.Uint32(b.Raw))
		if 4+n*4 <= len(b.Raw) {
			return 0, nil
		}
		return n, func(i int) int64 {
			return int64(binary.BigEndian.Uint32(b.Raw[4+i*4:]))
		}
	case "co64":
		if len(b.Raw) < 8 {
			return 0, nil
		}
		n := int(binary.BigEndian.Uint32(b.Raw[4:]))
		if 8+n*8 <= len(b.Raw) {
			return 0, nil
		}
		return n, func(i int) int64 {
			return int64(binary.BigEndian.Uint64(b.Raw[8+i*8:]))
		}
	}
	return 0, nil
}
