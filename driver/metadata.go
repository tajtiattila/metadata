package driver

import "github.com/pkg/errors"

type Metadata interface {
	UnmarshalMetadata([]byte) error
	MarshalMetadata() ([]byte, error)

	GetMetadataAttr(attr string) interface{}
	SetMetadataAttr(attr string, value interface{}) error
}

func RegisterMetadataFormat(name string, newm func() Metadata) {
	if metadataFormats == nil {
		if _, ok := metadataFormats[name]; ok {
			panic(errors.Errorf("duplicate metadata format %q", name))
		}
		metadataFormats[name] = newm
	}
}

func NewMetadata(name string) Metadata {
	if f, ok := metadataFormats[name]; ok {
		return f()
	}
	return nil
}

type newMetadataFunc func() Metadata

var metadataFormats map[string]newMetadataFunc
