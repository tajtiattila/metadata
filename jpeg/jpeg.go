package jpeg

import (
	"bytes"
	"errors"
	"io"
)

var (
	// ErrNotJpeg is returned if the file is not a jpeg file.
	ErrNotJpeg = errors.New("jpeg: missing start of image marker")

	// ErrTooLong is returned if the a chunk is too long to be written in an jpeg file.
	ErrTooLong = errors.New("jpeg: encoded length too long")
)

type Scanner struct {
	rr io.Reader

	buf  []byte
	r, w int // read and write position

	startChunk bool
	chunkLen   int // chunk bytes left

	p []byte

	scanState int

	err error

	// number of format errors encountered
	formatError int
}

const (
	scanStateBegin  = iota
	scanStateNormal // before start of scan
	scanStateScan   // start of scan seen
)

func NewScanner(r io.Reader) (*Scanner, error) {
	j := &Scanner{
		rr:  r,
		buf: make([]byte, 4096),
	}
	n, err := io.ReadAtLeast(j.rr, j.buf, 2)
	if err != nil {
		if err == io.EOF {
			// no bytes were read
			return nil, io.ErrUnexpectedEOF
		}
		return nil, err
	}
	if j.buf[0] != 0xff || j.buf[1] != 0xd8 {
		return nil, ErrNotJpeg
	}
	j.r, j.w = 0, n
	return j, nil
}

// Next() reads the next section
func (j *Scanner) Next() bool {
	if j.err != nil {
		return false
	}

	switch j.scanState {
	case scanStateBegin:
		// start of image
		j.p, j.r = j.buf[:2], 2
		j.scanState = scanStateNormal
		return true
	case scanStateScan:
		// start of scan seen
		return false
	}

	j.p = nil
	j.startChunk = false

	// process remaining chunk data
	if j.chunkLen > 0 {
		min := j.chunkLen
		if len(j.buf) < min {
			min = len(j.buf)
		}
		n, err := io.ReadAtLeast(j.rr, j.buf, min)
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		if err != nil {
			j.err = err
			return false
		}
		j.p = j.buf[:min]
		j.r, j.w = min, n
		j.chunkLen -= min
		return true
	}

	j.chunkLen = 0

	// readAhead is the number of bytes enough to recognise APP segments
	// JFIF  9 bytes: FF E0 .. .. 'J' 'F' 'I' 'F' 00
	// JFXX  9 bytes: FF E0 .. .. 'J' 'F' 'X' 'X' 00
	// EXIF 10 bytes: FF E1 .. .. 'E' 'x' 'i' 'f' 00 00
	const readAhead = 32

	// fill buffer until there is enough data to return,
	// or there is an error or EOF
	for j.err == nil && j.r+readAhead > j.w {
		if j.r != 0 {
			j.r, j.w = 0, copy(j.buf, j.buf[j.r:j.w])
		}

		n, err := j.rr.Read(j.buf[j.w:])
		j.w += n
		j.err = err
	}

	n := j.w - j.r
	if n < 4 {
		// no room for useful data left
		j.formatError++
		j.p = j.buf[j.r:j.w]
		j.r, j.w = 0, 0
		return len(j.p) != 0
	}

	// there is still data left
	if j.err == io.EOF {
		j.err = nil
	}

	// find next marker
	i := nextMarker(j.buf[j.r:j.w])
	if i > 0 {
		// no marker in buffer or
		// there is padding before the marker.
		// Return bytes to skip
		j.p = j.buf[j.r : j.r+i]
		j.r += i
	} else {
		// found marker at j.buf[j.r] with bytes:
		// 0xff marker sizehi sizelo
		if j.buf[j.r+1] == 0xda {
			// start of scan, we're done
			j.p = nil
			j.scanState = scanStateScan
			return false
		}
		l := chunkLen(j.buf[j.r:])
		if l == -1 {
			// invalid chunk length: skip marker and size
			j.formatError++
			s := j.r + 4
			j.r += 4
			j.p = j.buf[s:j.r]
			if j.r == j.w {
				j.r, j.w = 0, 0
			}
			return true
		}
		j.startChunk = true
		if j.r+l <= j.w {
			j.p = j.buf[j.r : j.r+l]
			j.r += l
		} else {
			j.p = j.buf[j.r:j.w]
			j.chunkLen = l - len(j.p)
			j.r, j.w = 0, 0
		}
	}

	return true
}

// StartChunk returns true if the last call to Next()
// found a chunk in the stream.
func (j *Scanner) StartChunk() bool {
	return j.startChunk
}

// ReadChunk reads the current chunk data the into a new slice
// after calling Next.
// If no new chunk has been found, it returns the padding between
// chunks.
func (j *Scanner) ReadChunk() ([]byte, error) {
	if j.err != nil {
		return nil, j.err
	}

	l := len(j.p) + j.chunkLen
	p := make([]byte, l)
	n := copy(p, j.p)
	j.p = nil

	for j.err == nil && j.chunkLen > 0 {
		var m int

		if j.chunkLen > len(j.buf) {
			// read large chunk directly into p
			m, j.err = io.ReadFull(j.rr, p[n:])
		} else {
			// read into buffer
			var buffered int
			buffered, j.err = io.ReadAtLeast(j.rr, j.buf, j.chunkLen)
			m = copy(p[n:], j.buf[:buffered])
			j.r, j.w = m, buffered
		}

		n += m
		j.chunkLen -= m
		if m == 0 && j.err == nil {
			j.err = io.ErrUnexpectedEOF
		}
	}

	return p[:n], j.err
}

// Len returns the currently available Bytes in Scanner.
func (j *Scanner) Len() int {
	return len(j.p)
}

// Bytes returns the most recent byte slice scanned after calling Next.
// The returned slice must not be modified.
// It is valid until the next call of Next or ReadChunk.
func (j *Scanner) Bytes() []byte {
	return j.p
}

// Reader returns a reader for the data remaining in the underlying reader.
func (j *Scanner) Reader() io.Reader {
	n := j.w - j.r
	if n == 0 {
		return j.rr
	}
	return io.MultiReader(bytes.NewReader(j.buf[j.r:j.w]), j.rr)
}

// nextMarker scans for the next marker.
// It returns either the marker position or an
// index near p
func nextMarker(p []byte) int {
	// search through p omitting the last 2 bytes
	// to simplify checking markers with content
	n := len(p) - 2
	if n < 0 {
		panic("nextMarker needs at least 2 bytes")
	}
	for i := 0; i < n; i++ {
		a, b := p[i], p[i+1]
		if a == 0xff && b != 0xff && b != 0x00 {
			if 0xd0 <= b && b <= 0xd9 {
				// marker with no content
				// NB: these should not appear here
				// because 0xff 0xd8 (SOI) should have been already seen
				// 0xd0-0xd7 (RST) and 0xd9 (EOI) should appear only after SOS
				continue
			}
			return i
		}
	}
	return n
}

// Err() returns any error encountered during Next()
func (j *Scanner) Err() error {
	return j.err
}

// chunkLen returns the number of bytes in the current chunk
// It returns -1 if the chunk is not valid.
func chunkLen(p []byte) int {
	if len(p) < 4 {
		return -1
	}
	l := int(p[2])<<8 + int(p[3])
	if l < 2 {
		// invalid chunk
		return -1
	}
	return l + 2 // length with marker
}

func WriteChunk(w io.Writer, marker byte, chunkdata []byte) error {
	n := len(chunkdata) + 4
	if n > 65535 {
		return ErrTooLong
	}

	var buf [4]byte
	buf[0] = 0xff
	buf[1] = marker
	buf[2] = byte(uint32(n) >> 8)
	buf[3] = byte(n)
	n, err := w.Write(buf[:])
	if err != nil {
		return err
	}
	if n != 4 {
		return io.ErrShortWrite
	}

	n, err = w.Write(chunkdata)
	if err != nil {
		return err
	}
	if n != len(chunkdata) {
		return io.ErrShortWrite
	}
	return nil
}
