package exif

import (
	"bytes"
	"image"
	_ "image/jpeg"
	"io"
	"os"
	"testing"
)

func TestCopy(t *testing.T) {
	fns := []string{
		"coffee-sf.jpg",
		"gocon-tokyo.jpg",
		"sub.jpg",
	}
	for _, fn := range fns {
		testCopy(t, fn)
	}
}

func testCopy(t *testing.T, fn string) {
	t.Log(fn)

	err := os.MkdirAll("../testdata/output", 0777)
	if err != nil {
		t.Fatal(err)
	}

	fi, err := os.Open("../testdata/" + fn)
	if err != nil {
		t.Fatalf("open source error: %v", err)
	}
	defer fi.Close()

	tbuf := new(bytes.Buffer)
	x, err := Decode(io.TeeReader(fi, tbuf))
	if err != nil {
		t.Fatalf("exif decode error: %v", err)
	}

	enc := new(bytes.Buffer)
	err = Copy(enc, io.MultiReader(tbuf, fi), x)
	if err != nil {
		t.Errorf("exif copy error: %v", err)
	}

	fo, err := os.Create("../testdata/output/" + fn)
	if err != nil {
		t.Fatalf("open destination error: %v", err)
	}
	defer fo.Close()

	_, err = fo.Write(enc.Bytes())
	if err != nil {
		t.Errorf("write destination error: %v", err)
	}

	// check if image validity through image/jpeg decoder
	_, _, err = image.Decode(enc)
	if err != nil {
		t.Errorf("write destination error: %v", err)
	}
}
