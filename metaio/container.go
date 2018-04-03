package metaio

import (
	"errors"
	"io"
)

func RegisterContainerFormat(name, magic string, format ContainerFormat) {

	containerFormats = append(containerFormats, containerFormat{
		name:   name,
		magic:  magic,
		format: format,
	})

	if len(magic) > containerPeekLen {
		containerPeekLen = len(magic)
	}
}

type ContainerFormat interface {
	Scan(r io.Reader, f func(name string, data []byte)) (map[string]interface{}, error)
	WriteWithMeta(w io.Writer, r io.Reader, m []EncodedMeta) error
}

type containerFormat struct {
	name, magic string

	format ContainerFormat
}

var containerFormats []containerFormat

var containerPeekLen int

var ErrUnknownFormat = errors.New("metadata: unknown content format")

func ContainerPeekLen() int { return containerPeekLen }

func GetContainerFormat(prefix []byte) (ContainerFormat, string) {
	for _, cf := range containerFormats {
		if isMagic(prefix, cf.magic) {
			return cf.format, cf.name
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

type EncodedMeta struct {
	Name string // metadata name, such as "exif" or "xmp"

	Bytes []byte // encoded metadata
}
