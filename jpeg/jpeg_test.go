package jpeg

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"
)

func TestScanner(t *testing.T) {
	fns := []string{
		"coffee-sf.jpg",
		"gocon-tokyo.jpg",
		"sub.jpg",
	}
	for _, fn := range fns {
		t.Log(fn)
		p, err := ioutil.ReadFile("../testdata/" + fn)
		if err != nil {
			t.Error(err)
			continue
		}
		testScannerBytes(t, p)
		testScannerChunks(t, p)
	}
}

func testScannerBytes(t *testing.T, p []byte) {
	t.Log("testScannerBytes")

	q := make([]byte, len(p))

	s, err := NewScanner(bytes.NewReader(p))
	if err != nil {
		t.Error("testScannerBytes newJpegScanner:", err)
		return
	}
	i := 0
	for s.Next() {
		if s.Len() == 0 {
			t.Error("testScannerBytes got 0 bytes")
			return
		}

		t.Logf("%-5v %x", s.StartChunk(), s.Bytes())
		n := copy(q[i:], s.Bytes())
		if n == 0 {
			t.Error("testScannerBytes too many bytes")
			return
		}
		i += n
	}
	if err := s.Err(); err != nil {
		t.Error("testScannerBytes finish", err)
		return
	}
	if _, err = io.ReadFull(s.Reader(), q[i:]); err != nil {
		t.Error(err)
		return
	}

	if !bytes.Equal(p, q) {
		t.Error("bytes scanned with jpegScanner.Bytes differ from source")
	}
}

func testScannerChunks(t *testing.T, p []byte) {
	t.Log("testScannerChunks")

	var chunks [][]byte

	s, err := NewScanner(bytes.NewReader(p))
	if err != nil {
		t.Error("testScannerChunks newJpegScanner:", err)
		return
	}
	for s.Next() {
		chunk, err := s.ReadChunk()
		if err != nil {
			t.Error("testScannerChunks ReadChunk:", err)
			return
		}
		if len(chunk) == 0 {
			t.Error("testScannerChunks got empty chunk")
			continue
		}
		if s.StartChunk() {
			if len(chunk) < 4 ||
				chunk[0] != 0xff || chunk[1] == 0 || chunk[1] == 0xff {
				t.Error("testScannerChunks invalid chunk %x:", chunk)
				return
			}
			l := int(chunk[2])<<8 + int(chunk[3])
			if l+2 != len(chunk) {
				t.Error("testScannerChunks chunk len: want %v got %v", l+2, len(chunk))
			}
		}
		t.Logf("%-5v %x", s.StartChunk(), chunk)
		chunks = append(chunks, chunk)
	}
	if err := s.Err(); err != nil {
		t.Error("testScannerChunks finish", err)
		return
	}

	last, err := ioutil.ReadAll(s.Reader())
	if err != nil {
		t.Error("testScannerChunks read last bits:", err)
	}
	chunks = append(chunks, last)

	q := bytes.Join(chunks, nil)

	if !bytes.Equal(p, q) {
		t.Error("testScannerChunks scanned bytes differ from source")
	}
}
