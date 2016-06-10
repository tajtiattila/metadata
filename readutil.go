package metadata

import (
	"bytes"
	"errors"
	"io"
)

// prefixReader returns r so that it has pfx unread from it
func prefixReader(pfx []byte, r io.Reader) io.Reader {
	rs, ok := r.(io.ReadSeeker)
	if ok {
		// try to seek back through prefix
		_, err := rs.Seek(-int64(len(pfx)), 1)
		if err == nil {
			return rs
		}
		return &pfxReadSeeker{pfx, rs, 0}
	}
	return io.MultiReader(bytes.NewReader(pfx), rs)
}

type pfxReadSeeker struct {
	pfx []byte
	rs  io.ReadSeeker

	off int
}

func (r *pfxReadSeeker) Read(p []byte) (n int, err error) {
	var m int
	if r.off < len(r.pfx) {
		m = copy(p, r.pfx[r.off:])
		r.off, p = r.off+m, p[m:]
		if len(p) == 0 {
			return m, nil
		}
	}
	n, err = r.rs.Read(p)
	return n + m, err
}

func (r *pfxReadSeeker) Seek(offset int64, whence int) (int64, error) {
	if whence != 1 || offset < 0 {
		return 0, errors.New("pfxReadSeeker.Seek supports only relative forward seek")
	}

	npfx := len(r.pfx) - r.off
	if npfx > 0 {
		if offset <= int64(npfx) {
			r.off += int(offset)
			return int64(r.off), nil
		}
		r.off, offset = r.off+npfx, offset-int64(npfx)
	}

	return r.rs.Seek(offset, whence)
}

type atReadSeeker struct {
	off int64
	io.ReaderAt
}

func (a *atReadSeeker) Read(p []byte) (n int, err error) {
	n, err = a.ReadAt(p, a.off)
	a.off += int64(n)
	return
}

var errWhence = errors.New("Seek: invalid whence")
var errSeekEnd = errors.New("Seek: atReadSeeker doesn't support seeking from end")
var errSeekOffset = errors.New("Seek: invalid offset")

func (a *atReadSeeker) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case 0:
		// pass
	case 1:
		offset += a.off
	case 2:
		s, ok := a.ReaderAt.(sizer)
		if !ok {
			return a.off, errSeekEnd
		}
		offset += s.Size()
	default:
		return a.off, errWhence
	}
	if offset < 0 {
		return a.off, errSeekOffset
	}
	a.off = offset
	return a.off, nil
}

type sizer interface {
	Size() int64
}
