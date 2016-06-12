package exif

import (
	"bytes"
	"image"
	_ "image/jpeg"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/tajtiattila/metadata/testutil"
)

func TestCopy(t *testing.T) {
	for _, fn := range testutil.MediaFileNames(t, "image/jpeg") {
		testCopy(t, fn)
	}
}

func testCopy(t *testing.T, fn string) {
	fi, err := os.Open(fn)
	if err != nil {
		t.Fatalf("open source error: %v", err)
	}
	defer fi.Close()

	tbuf := new(bytes.Buffer)
	x, err := Decode(io.TeeReader(fi, tbuf))
	if err != nil {
		if err == NotFound {
			return
		}
		t.Fatalf("exif decode error: %v", err)
	}

	_, _, err = image.Decode(bytes.NewReader(tbuf.Bytes()))
	if err != nil {
		// jpeg is corrupt, don't bother trying to copy it
		return
	}

	enc := new(bytes.Buffer)
	err = Copy(enc, io.MultiReader(tbuf, fi), x)
	if err != nil {
		t.Errorf("%s exif copy error: %v", fn, err)
	}

	destDir, err := ioutil.TempDir("", "metadata_exif_test")
	if err != nil {
		t.Fatal("can't create temp dir:", err)
	}
	defer os.RemoveAll(destDir)

	fo, err := os.Create(filepath.Join(destDir, filepath.Base(fn)))
	if err != nil {
		t.Fatalf("open destination error: %v", err)
	}
	defer fo.Close()

	_, err = fo.Write(enc.Bytes())
	if err != nil {
		t.Errorf("%s write destination error: %v", fn, err)
	}

	// check if image validity through image/jpeg decoder
	_, _, err = image.Decode(enc)
	if err != nil {
		t.Errorf("%s write destination error: %v", fn, err)
	}
}
