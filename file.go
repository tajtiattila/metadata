package metadata

import (
	"io"
	"io/ioutil"
)

// FileOp represents a change that should be applied to a stream.
//
// The relation between Size and len(Data) tells what operation needs
// to be performed. If they are the same, the source bytes need to be overwritten,
// otherwise bytes need to be inserted or removed from the source stream.
type FileOp struct {
	Offset int64  // file offset of this change
	Size   int    // number of bytes in the source to skip at Offset
	Data   []byte // data to insert at Offset
}

// FileMod represents modification, which is a (possibly empty)
// set of operations to be applied to a file.
//
// Elements of FileMod must be ordered with increasing Offset and must not overlap.
type FileMod []FileOp

// Copy copies r to w with ops applied.
// It panics if ops is invalid.
func (ops FileMod) Copy(w io.Writer, r io.Reader) (written int64, err error) {
	if !ops.Valid() {
		panic("FileMod invalid")
	}

	var read int64
	for _, o := range ops {
		if ncopy := o.Offset - read; ncopy > 0 {
			// copy bytes before o
			n, err := io.CopyN(w, r, ncopy)
			read += n
			written += n
			if err != nil {
				if err == io.EOF {
					// ops invalid or do not match with this source
					err = io.ErrUnexpectedEOF
				}
				return written, err
			}
		}

		// write data to r
		if len(o.Data) != 0 {
			n, err := w.Write(o.Data)
			written += int64(n)
			if err != nil {
				return written, err
			}
			if n != len(o.Data) {
				return written, io.ErrShortWrite
			}
		}

		// discard overwritten/deleted bytes from r
		if o.Size != 0 {
			n, err := io.CopyN(ioutil.Discard, r, int64(o.Size))
			read += n
			if err != nil {
				return written, err
			}
		}
	}

	// copy remaining bytes, if any
	n, err := io.Copy(w, r)
	written += n
	return written, err
}

// Reader returns an io.Reader that applies ops to r.
// It panics if ops is invalid.
func (ops FileMod) Reader(r io.Reader) io.Reader {
	if !ops.Valid() {
		panic("FileMod invalid")
	}

	if len(ops) == 0 {
		return r
	}
	return &opreader{r: r, ops: ops, tmp: make([]byte, 4096)}
}

// Valid reports if ops has properly ordered non-overlaping elements.
func (ops FileMod) Valid() bool {
	var off int64
	for _, o := range ops {
		if o.Offset < off {
			return false
		}
		off = o.Offset + int64(o.Size)
	}
	return true
}

type opreader struct {
	r   io.Reader
	ops []FileOp

	ro int64

	i  int // current op index
	do int // index into current op data

	tmp []byte
}

func (r *opreader) Read(p []byte) (n int, err error) {
	for r.i < len(r.ops) {
		o := r.ops[r.i]

		// read bytes from source before offset
		if ncopy := o.Offset - r.ro; ncopy > 0 {
			q := p
			if int64(len(p)) > ncopy {
				q = q[:int(ncopy)]
			}
			m, err := r.r.Read(q)
			p, n, r.ro = p[m:], n+m, r.ro+int64(m)
			if err != nil {
				return n, err
			}
		}

		if r.ro < o.Offset {
			// offset not yet reached
			return n, nil
		}

		// yield op data
		m := copy(p, o.Data[r.do:])
		p, n, r.do = p[m:], n+m, r.do+m

		if r.do < len(o.Data) {
			return n, nil
		}

		// discard skipped bytes from source
		const skipbufsize = 4096
		tmp := r.tmp
		if len(tmp) < len(p) {
			tmp = p
		}

		end := o.Offset + int64(o.Size)
		var err error
		for r.ro < end && err == nil {
			m := int(end - r.ro)
			if len(tmp) < m {
				m = len(tmp)
			}
			m, err = r.r.Read(tmp[:m])
			r.ro += int64(m)
		}
		if r.ro == end {
			r.i++
			r.do = 0
		}
		if err != nil {
			return n, err
		}
	}

	// past last op
	m, err := r.r.Read(p)
	return n + m, err
}

func zerofill(p []byte, maxfill int) int {
	n := maxfill
	if len(p) < n {
		n = len(p)
	}
	for i := range p {
		p[i] = 0
	}
	return n
}
