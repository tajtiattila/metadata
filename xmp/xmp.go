package xmp

import (
	"encoding/xml"
	"errors"
	"io"

	"github.com/tajtiattila/xmlutil"
)

type Meta struct {
	Doc xmlutil.Document

	cache map[xml.Name]*xmlutil.Node
}

func Decode(r io.Reader) (*Meta, error) {
	m := new(Meta)
	if err := xml.NewDecoder(r).Decode(&m.Doc); err != nil {
		return nil, err
	}

	if err := m.cacheRdfs(); err != nil {
		return nil, err
	}

	return m, nil
}

const (
	metans = "adobe:ns:meta/"
	rdfns  = "http://www.w3.org/1999/02/22-rdf-syntax-ns#"
)

var ErrInvalidXMLFormat = errors.New("xmp: invalid XML format")

func (m *Meta) cacheRdfs() error {
	root := m.Doc.Node
	if root.Name == xmlName(metans, "xmpmeta") {
		if len(root.Child) != 1 {
			return ErrInvalidXMLFormat
		}
		root = root.Child[0]
	}

	if root.Name != xmlName(rdfns, "RDF") {
		return ErrInvalidXMLFormat
	}

	m.cache = make(map[xml.Name]*xmlutil.Node)
	for _, n := range root.Child {
		m.cacheNodes(n)
	}

	return nil
}

func (m *Meta) cacheNodes(n *xmlutil.Node) {
	if n.Name != xmlName(rdfns, "Description") {
		return
	}

	for _, c := range n.Child {
		m.cache[c.Name] = c
	}
}

func (m *Meta) String(f StringFunc) (value string, ok bool) {
	return f(m)
}

func (m *Meta) Int(f IntFunc) (value int, ok bool) {
	return f(m)
}

func (m *Meta) Float64(f Float64Func) (value float64, ok bool) {
	return f(m)
}
