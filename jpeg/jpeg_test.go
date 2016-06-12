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
		testScannerChunks(t, p)
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

// testScannerChunks tests if using Scanner.ReadChunk on
// the bytes of p yields the same bytes as the source.
func testScannerChunks(t *testing.T, p []byte) {
	t.Log("testScannerChunks")

	var chunks [][]byte

	s, err := NewScanner(bytes.NewReader(p))
	if err != nil {
		t.Error("NewScanner error:", err)
		return
	}
	for s.Next() {
		chunk, err := s.ReadChunk()
		if err != nil {
			t.Error("ReadChunk error:", err)
			return
		}
		if len(chunk) == 0 {
			t.Error("error: testScannerChunks got empty chunk")
			continue
		}
		if s.StartChunk() {
			if len(chunk) < 4 ||
				chunk[0] != 0xff || chunk[1] == 0 || chunk[1] == 0xff {
				t.Error("error: testScannerChunks invalid chunk %x:", chunk)
				return
			}
			l := int(chunk[2])<<8 + int(chunk[3])
			if l+2 != len(chunk) {
				t.Error("error: testScannerChunks chunk len: want %v got %v", l+2, len(chunk))
			}
		}
		t.Logf("%-5v %4d %.32x", s.StartChunk(), len(chunk), chunk)
		chunks = append(chunks, chunk)
	}
	if err := s.Err(); err != nil {
		t.Error("testScannerChunks finish error:", err)
		return
	}

	last, err := ioutil.ReadAll(s.Reader())
	if err != nil {
		t.Error("testScannerChunks read last bits error:", err)
	}
	chunks = append(chunks, last)

	q := bytes.Join(chunks, nil)

	if !bytes.Equal(p, q) {
		t.Error("error: estScannerChunks scanned bytes differ from source")
	}
}

func dumpBytes(w io.Writer, p []byte) {
	for i := 0; i < len(p); i += 32 {
		fmt.Fprintf(w, "% .32x\n", p[i:])
	}
}
