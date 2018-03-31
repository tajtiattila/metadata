package driver

import (
	"bytes"
	"errors"
	"io"
)

type Container interface {
	Parse(r io.Reader) error
	WriteTo(w io.Writer) error
}

func RegisterContainerFormat(name, magic string, newc func() Container) {
	containerFormats = append(containerFormats, containerFormat{
		name:  name,
		magic: magic,
		new:   newc,
	})
}

type containerFormat struct {
	name, magic string

	new func() Container
}

var containerFormats []containerFormat

var ErrUnknownFormat = errors.New("metadata: unknown content format")

func NewContainer(r io.Reader) (Container, string, error) {

	const prefixLen = 16
	buf := make([]byte, prefixLen)

	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, "", err
	}

	c, cname := getContainer(buf)
	if c == nil {
		return nil, "", ErrUnknownFormat
	}

	var rr io.Reader

	rs, isSeeker := r.(io.ReadSeeker)
	if isSeeker {
		if _, err := rs.Seek(0, io.SeekStart); err != nil {
			return nil, "", err
		}
		rr = r
	} else {
		rr = io.MultiReader(bytes.NewReader(buf), r)
	}

	err := c.Parse(rr)
	if err != nil {
		return nil, "", err
	}

	return c, cname, nil
}

func getContainer(prefix []byte) (Container, string) {
	for _, cf := range containerFormats {
		if isMagic(prefix, cf.magic) {
			return cf.new(), cf.name
		}
	}
	return nil, ""
}

func isMagic(prefix []byte, magic string) bool {
	if len(prefix) < len(magic) {
		return false
	}
	for i := 0; i < len(magic); i++ {
		if magic[i] != '?' && magic[i] != prefix[i] {
			return false
		}
	}
	return true
}
