package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func MediaRoot(t *testing.T) string {
	if v := os.Getenv("MEDIA_TEST"); v != "" {
		return v
	}
	const testMedia = "github.com/tajtiattila/test-media"
	for _, x := range filepath.SplitList(os.Getenv("GOPATH")) {
		p := filepath.Join(x, "src", testMedia)
		if s, err := os.Stat(p); err == nil && s.Mode().IsDir() {
			return p
		}
	}
	t.Skip("test-media not found")
	panic("unreachable")
}

// MediaFileInfos returns test file paths having mimetype.
func MediaFileInfos(t *testing.T) []FileInfo {
	root := MediaRoot(t)

	var fi []FileInfo

	pth := filepath.Join(root, "exiftool.json")
	f, err := os.Open(pth)
	if err != nil {
		t.Skipf("%s not found", pth)
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&fi); err != nil {
		t.Skipf("%s decode error %v", pth, err)
	}

	for _, e := range fi {
		fn, ok := e.String("SourceFile")
		if !ok {
			continue
		}
		e["SourceFile"] = filepath.Join(root, fn)
	}

	return fi
}

// MediaFileNames returns test file paths having mimetype.
func MediaFileNames(t *testing.T, mimetype string) []string {
	var files []string
	for _, e := range MediaFileInfos(t) {
		fn, ok := e.String("SourceFile")
		if !ok {
			continue
		}

		if mimetype != "" {
			if mt, ok := e.String("MIMEType"); !ok || mt != mimetype {
				continue
			}
		}

		files = append(files, fn)
	}

	if len(files) == 0 {
		t.Skip("no test files not found")
	}
	return files
}

type FileInfo map[string]interface{}

func (e FileInfo) Int(n string) (v int, ok bool) {
	x, ok := e[n]
	if !ok {
		return
	}
	w, ok := x.(float64)
	return int(w), ok
}

func (e FileInfo) Float64(n string) (v float64, ok bool) {
	x, ok := e[n]
	if !ok {
		return
	}
	v, ok = x.(float64)
	return v, ok
}

func (e FileInfo) String(n string) (v string, ok bool) {
	x, ok := e[n]
	if !ok {
		return
	}
	v, ok = x.(string)
	return v, ok
}
