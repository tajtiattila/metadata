package metadata

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"
	"math/rand"
	"testing"
)

func TestFileOps(t *testing.T) {
	const siz = 1 << 18
	p, err := ioutil.ReadAll(rr(siz))
	if err != nil {
		t.Fatal(err)
	}

	bj := func(v ...[]byte) []byte {
		return bytes.Join(v, nil)
	}

	ofs, skip := 1<<10, 1<<7
	testFileOps(t,
		bj(p[:ofs], p[ofs+skip:]),
		rr(siz),
		FileOp{int64(ofs), skip, nil})

	ofs, skip = 100123, 987
	ins := []byte("foobar")
	testFileOps(t,
		bj(p[:ofs], ins, p[ofs+skip:]),
		rr(siz),
		FileOp{int64(ofs), skip, ins})

	ofs, skip = 1<<16-3, 0
	ins = bytes.Repeat([]byte("baz"), 1<<12)
	testFileOps(t,
		bj(p[:ofs], ins, p[ofs+skip:]),
		rr(siz),
		FileOp{int64(ofs), skip, ins})
}

func testFileOps(t *testing.T, want []byte, r io.Reader, ops ...FileOp) {
	const logRead = false
	xr := func(tag string, r io.Reader) io.Reader {
		if logRead {
			return &logReader{"cmp", t, r, 0}
		} else {
			return r
		}
	}

	// set up pipe to test unused Copy implementation
	pr, pw := io.Pipe()
	defer pw.Close()
	xch := make(chan []byte)
	go func() {
		buf := new(bytes.Buffer)
		_, err := FileOps(ops).Copy(buf, pr)
		if err != nil {
			t.Error(err)
		}
		xch <- buf.Bytes()
	}()

	io.Copy(ioutil.Discard, xr("cmp", bytes.NewReader(want)))

	r = xr("raw", io.TeeReader(r, pw))
	read, err := ioutil.ReadAll(xr("ops", FileOps(ops).Reader(r)))
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(want, read) {
		t.Error("FileOps.Reader data mismatch")
	}

	pw.Close()
	xcopy := <-xch
	if !bytes.Equal(want, xcopy) {
		t.Error("FileOps.xCopy data mismatch")
	}
}

func rr(n int64) io.Reader {
	return &io.LimitedReader{new(randReader), n}
}

type logReader struct {
	tag string
	t   *testing.T
	r   io.Reader
	off int64
}

func (r *logReader) Read(p []byte) (n int, err error) {
	n, err = r.r.Read(p)
	j := 0
	for i := range p[:n] {
		if i != j && (int(r.off)+i)&15 == 0 {
			r.t.Logf("%s %6x % x", r.tag, r.off+int64(j), p[j:i])
			j = i
		}
	}
	if j < n {
		r.t.Logf("%s %6x % x", r.tag, r.off+int64(j), p[j:n])
	}
	r.off += int64(n)
	return n, err
}

type randReader struct {
	off int
	buf [4]byte
	rnd *rand.Rand
}

func (r *randReader) Read(p []byte) (n int, err error) {
	for i := range p {
		p[i] = r.nextByte()
	}
	return len(p), nil
}

func (r *randReader) nextByte() byte {
	if r.off == 0 {
		if r.rnd == nil {
			r.rnd = rand.New(rand.NewSource(0))
		}
		binary.BigEndian.PutUint32(r.buf[:], r.rnd.Uint32())
	}
	o := r.off
	r.off = (r.off + 1) % 3
	return r.buf[o]
}
