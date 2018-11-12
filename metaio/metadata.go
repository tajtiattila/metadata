package metaio

import "github.com/pkg/errors"

// Metadata represents a set of metadata attributes.
type Metadata interface {
	FormatName() string

	GetAttr(attr string) interface{}

	SetAttr(attr string, value interface{}) error
	DelAttr(attr string) error
}

// IOMetadata is Metadata that
// can marshal and unmarshal itself in its own format.
type IOMetadata interface {
	Metadata

	UnmarshalMetadata([]byte) error
	MarshalMetadata() ([]byte, error)
}

func RegisterMetadataFormat(name string, newm func() Metadata) {
	if metadataFormats == nil {
		metadataFormats = make(map[string]newMetadataFunc)
	}

	if _, ok := metadataFormats[name]; ok {
		panic(errors.Errorf("duplicate metadata format %q", name))
	}
	metadataFormats[name] = newm
}

func NewMetadata(name string) Metadata {
	if f, ok := metadataFormats[name]; ok {
		return f()
	}
	return nil
}

type newMetadataFunc func() Metadata

var metadataFormats map[string]newMetadataFunc
