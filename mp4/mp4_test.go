package mp4_test

import (
	"os"
	"testing"

	"github.com/tajtiattila/metadata/mp4"
	"github.com/tajtiattila/metadata/testutil"
)

func TestParse(t *testing.T) {
	fileList := testutil.MediaFileInfos(t)

	for _, e := range fileList {
		testParse(t, e)
	}
}

func testParse(t *testing.T, e testutil.FileInfo) {
	if _, ok := e["Error"]; ok {
		// exiftool found error parsing
		return
	}

	fn, ok := e.String("SourceFile")
	if !ok {
		return
	}

	if tp, ok := e.String("FileType"); !ok || tp != "MP4" {
		return
	}

	ex, ok := e.Int("ImageWidth")
	if !ok {
		return
	}

	ey, ok := e.Int("ImageHeight")
	if !ok {
		return
	}

	f, err := os.Open(fn)
	if err != nil {
		t.Errorf("Open %q error: %v", fn, err)
		return
	}
	defer f.Close()

	movie, err := mp4.Parse(f)
	if err != nil {
		t.Errorf("mp4.Parse of %q error: %v", fn, err)
		return
	}

	sx, sy, err := movie.FrameSize()
	if err != nil {
		t.Errorf("movie FrameSize error: %v", err)
		return
	}

	t.Logf("%q %vx%v\n", fn, sx, sy)

	if ex != sx || ey != sy {
		t.Errorf("got size %vx%v, want %vx%v", sx, sy, ex, ey)
	}
}
