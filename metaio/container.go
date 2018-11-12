package metaio

import (
	"errors"
	"io"
)

// Scan scans for encoded metadata in a registered container format.
// The returned Metadata contains only implicit metadata specific to the container.
// The string returned is the format name used during container format registration.
// Format registration is typically done by an init function in the codec-specific package.
//
// Scan calls f with the name and encoded bytes
// of every metadata segment recognised by the container format.
//
// If r is an io.ReadSeeker, it may be used to seek in the source.
func Scan(r io.Reader, f func(name string, data []byte)) (Metadata, string, error) {
	return nil, "", nil
}

// Container represents a metadata container.
type Container interface {
	// Metadata returns implicit metadata specific to the container,
	// such as image dimensions for an image container.
	Metadata() Metadata

	// RawMeta returns encoded metadata in the container.
	RawMeta() []RawMeta

	// SetRawMeta sets metadata to r.
	SetRawMeta(r []RawMeta)

	// AsReadSeeker returns a new ReadSeeker for the Container
	// having new metadata set with SetRawMeta.
	AsReadSeeker() io.ReadSeeker
}

// RawMeta represents encoded metadata.
type RawMeta struct {
	Name  string // metadata format name
	Bytes []byte // marshaled metadata
}

// Parse parses a metadata container and registered container format,
// recording file structure.
// Format registration is typically done by an init function in the codec-specific package.
func Parse(rs io.ReadSeeker) {
}

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

type scanFunc func(r io.Reader, f func(name string, data []byte)) error

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
