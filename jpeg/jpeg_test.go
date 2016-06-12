package jpeg

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	"github.com/tajtiattila/metadata/testutil"
)

func TestScanner(t *testing.T) {
	for _, fn := range testutil.MediaFileNames(t, "image/jpeg") {
		t.Log(fn)
		p, err := ioutil.ReadFile(fn)
		if err != nil {
			t.Error(err)
			continue
		}
		testScannerBytes(t, p)
		testScannerSegments(t, p)
	}
}

// testScannerBytes tests if using Scanner.Bytes on
// the bytes of p yields the same bytes as the source.
func testScannerBytes(t *testing.T, p []byte) {
	q := make([]byte, len(p))

	s, err := NewScanner(bytes.NewReader(p))
	if err != nil {
		t.Error("NewScanner error:", err)
		return
	}

	dump := new(bytes.Buffer)
	i := 0
	for s.Next() {
		if s.Len() == 0 {
			t.Errorf("error: testScannerBytes got 0 bytes\n%s", dump.Bytes())
			return
		}

		fmt.Fprintf(dump, "%6d %5d %v\n", i, s.Len(), s.StartChunk())
		dumpBytes(dump, s.Bytes())

		// check if we have the same bit in src
		part := p[i:]
		if s.Len() < len(part) {
			part = part[:s.Len()]
		}
		if !bytes.Equal(part, s.Bytes()) {
			t.Errorf("error: testScannerBytes mismatch at %d %.32x %.32x\n%s",
				i, part, s.Bytes(), dump.Bytes())
			return
		}

		n := copy(q[i:], s.Bytes())
		if n == 0 {
			t.Errorf("error: testScannerBytes too many bytes\n%s", dump.Bytes())
			return
		}
		i += n
	}
	if err := s.Err(); err != nil {
		t.Error("testScannerBytes finish error:", err)
		return
	}
	if _, err = io.ReadFull(s.Reader(), q[i:]); err != nil {
		t.Error(err)
		return
	}

	if !bytes.Equal(p, q) {
		t.Error("error: bytes scanned with jpegScanner.Bytes differ from source")
	}
}

// testScannerSegments tests if using Scanner.ReadSegment on
// the bytes of p yields the same bytes as the source.
func testScannerSegments(t *testing.T, p []byte) {
	t.Log("testScannerSegments")

	var segments [][]byte

	s, err := NewScanner(bytes.NewReader(p))
	if err != nil {
		t.Error("NewScanner error:", err)
		return
	}
	for s.Next() {
		seg, err := s.ReadSegment()
		if err != nil {
			t.Error("ReadSegment error:", err)
			return
		}
		if len(seg) == 0 {
			t.Error("error: testScannerSegments got empty segment")
			continue
		}
		if s.StartChunk() {
			if len(seg) < 4 ||
				seg[0] != 0xff || seg[1] == 0 || seg[1] == 0xff {
				t.Error("error: testScannerSegments invalid segment %x:", seg)
				return
			}
			l := int(seg[2])<<8 + int(seg[3])
			if l+2 != len(seg) {
				t.Error("error: testScannerSegments segment len: want %v got %v", l+2, len(seg))
			}
		}
		t.Logf("%-5v %4d %.32x", s.StartChunk(), len(seg), seg)
		segments = append(segments, seg)
	}
	if err := s.Err(); err != nil {
		t.Error("testScannerSegments finish error:", err)
		return
	}

	last, err := ioutil.ReadAll(s.Reader())
	if err != nil {
		t.Error("testScannerSegments read last bits error:", err)
	}
	segments = append(segments, last)

	q := bytes.Join(segments, nil)

	if !bytes.Equal(p, q) {
		t.Error("error: estScannerChunks scanned bytes differ from source")
	}
}

func dumpBytes(w io.Writer, p []byte) {
	for i := 0; i < len(p); i += 32 {
		fmt.Fprintf(w, "% .32x\n", p[i:])
	}
}
