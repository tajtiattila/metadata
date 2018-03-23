package png

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"io/ioutil"
)

type File struct {
	XMP []byte // raw XMP metadata
}

const pngHeader = "\x89PNG\r\n\x1a\n"

const xmpKeyword = "XML:com.adobe.xmp"

func Parse(r io.Reader) (*File, error) {
	d := decoder{
		r:   r,
		tmp: make([]byte, 1<<16),
	}
	if err := d.decode(); err != nil {
		return nil, err
	}
	f := &File{
		XMP: d.xmp,
	}
	return f, nil
}

type decoder struct {
	r   io.Reader
	tmp []byte

	xmp []byte
}

func (d *decoder) decode() error {
	_, err := io.ReadFull(d.r, d.tmp[:len(pngHeader)])
	if err != nil {
		return err
	}
	if string(d.tmp[:len(pngHeader)]) != pngHeader {
		return errors.New("not a png file")
	}
	for {
		// Read the length and chunk type.
		_, err := io.ReadFull(d.r, d.tmp[:8])
		if err != nil {
			return err
		}
		length := binary.BigEndian.Uint32(d.tmp[:4])
		switch string(d.tmp[4:8]) {
		case "iTXt":
			err := d.decodeiTXt(length)
			if err != nil {
				return err
			}
			if d.xmp != nil {
				return nil
			}
		default:
			d.skip(length + 4)
		}
	}
	return nil
}

func (d *decoder) skip(length uint32) error {
	// length and CRC
	l := int(length)
	if s, ok := d.r.(io.Seeker); ok {
		_, err := s.Seek(int64(l), io.SeekCurrent)
		return err
	}
	for l > 0 {
		n := l
		if n > len(d.tmp) {
			n = len(d.tmp)
		}
		_, err := io.ReadFull(d.r, d.tmp[:n])
		if err != nil {
			return err
		}
		l -= n
	}
	return nil
}

func (d *decoder) decodeiTXt(length uint32) error {
	l := int(length)

	// set up reader for chunk data
	chunkr := io.LimitReader(d.r, int64(l))

	// fill tmp with chunk data or chunk data prefix
	n := l
	if n > len(d.tmp) {
		n = len(d.tmp)
	}
	_, err := io.ReadFull(chunkr, d.tmp[:n])
	if err != nil {
		return err
	}
	l -= n

	h, nh, err := decodeiTXtHeader(d.tmp[:n])
	if err != nil {
		return err
	}

	// we care about only XMP
	if h.keyword != xmpKeyword {
		return d.skip(uint32(l + 4))
	}

	crc := crc32.NewIEEE()

	// set up reader for text data, calculating crc
	textr := io.TeeReader(
		io.MultiReader(
			bytes.NewReader(d.tmp[nh:n]),
			chunkr,
		), crc)

	if !(h.compression == 0 ||
		(h.compression == 1 && h.compressionMethod == 0)) {
		return errors.New("unsupported compression method")
	}

	if h.compression == 1 {
		x, err := zlib.NewReader(textr)
		if err != nil {
			return err
		}
		defer x.Close()
		textr = x
	}

	xmp, err := ioutil.ReadAll(textr)
	if err != nil {
		return nil
	}

	if h.compression == 1 {
		// read extra bytes after compressed data, if any
		if _, err := io.Copy(ioutil.Discard, chunkr); err != nil {
			return err
		}
	}

	// verify checksum
	if _, err := io.ReadFull(d.r, d.tmp[:4]); err != nil {
		return err
	}
	if binary.BigEndian.Uint32(d.tmp[:4]) != crc.Sum32() {
		return errors.New("invalid checksum")
	}

	d.xmp = xmp

	return nil
}

// https://www.w3.org/TR/PNG/#11iTXt
type iTXthdr struct {
	keyword           string
	compression       byte
	compressionMethod byte
	languageTag       string
	translatedKeyword string
}

func decodeiTXtHeader(p []byte) (h *iTXthdr, length int, err error) {
	d := itxtDec{src: p}

	h = &iTXthdr{
		keyword:           d.string(),
		compression:       d.byte(),
		compressionMethod: d.byte(),
		languageTag:       d.string(),
		translatedKeyword: d.string(),
	}

	if d.fail {
		return nil, 0, errors.New("invalid iTXt header")
	}

	return h, d.pos, nil
}

type itxtDec struct {
	src  []byte
	pos  int
	fail bool
}

func (d *itxtDec) string() string {
	if d.fail {
		return ""
	}

	i := bytes.IndexByte(d.src[d.pos:], 0)
	if i == -1 {
		d.fail = true
		return ""
	}

	p := d.pos
	n := d.pos + i
	d.pos = n + 1

	return string(d.src[p:n])
}

func (d *itxtDec) byte() byte {
	if d.fail || d.pos < len(d.src) {
		d.fail = true
		return 0
	}

	b := d.src[d.pos]
	d.pos++
	return b
}
