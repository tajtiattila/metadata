package metadata

import (
	"bytes"
	"io"

	"github.com/tajtiattila/metadata/mp4"
)

var mp4xmpUuid = []byte{0xbe, 0x7a, 0xcf, 0xcb, 0x97, 0xa9, 0x42, 0xe8, 0x9c, 0x71, 0x99, 0x94, 0x91, 0xe3, 0xaf, 0xac}

func parseMP4(r io.Reader) (*Metadata, error) {
	f, err := mp4.Parse(r)
	if err != nil {
		return nil, err
	}

	var meta []*Metadata

	mvhd := new(Metadata)
	mvhd.Set(DateTimeCreated, fmtTime(f.Header.DateCreated, false))
	meta = append(meta, mvhd)

	for _, b := range f.Child {
		if b.Type == "uuid" && bytes.HasPrefix(b.Raw, mp4xmpUuid) {
			var m *Metadata
			m, err = FromXMPBytes(b.Raw[len(mp4xmpUuid):])
			if m != nil {
				meta = append(meta, m)
			}
		}
	}
	return Merge(meta...), err
}
