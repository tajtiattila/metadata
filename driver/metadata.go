package driver

import "github.com/pkg/errors"

// Option is used in Metadata initialization.
type Option interface{}

// ImageSize is a metadata option stating image size.
type ImageSize struct {
	Width, Height int
}

type Metadata interface {
	MetadataName() string

	UnmarshalMetadata([]byte) error
	MarshalMetadata() ([]byte, error)

	GetMetadataAttr(attr string) interface{}
	SetMetadataAttr(attr string, value interface{}) error
	DeleteMetadataAttr(attr string) error
}

func RegisterMetadataFormat(name string, newm func(...Option) Metadata) {
	if metadataFormats == nil {
		metadataFormats = make(map[string]newMetadataFunc)
	}

	if _, ok := metadataFormats[name]; ok {
		panic(errors.Errorf("duplicate metadata format %q", name))
	}
	metadataFormats[name] = newm
}

func NewMetadata(name string, opt ...Option) Metadata {
	if f, ok := metadataFormats[name]; ok {
		return f(opt...)
	}
	return nil
}

type newMetadataFunc func(opt ...Option) Metadata

var metadataFormats map[string]newMetadataFunc
