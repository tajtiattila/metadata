package xmp

import (
	"encoding/xml"
	"io"
)

type Meta struct {
	XMLName xml.Name `xml:"adobe:ns:meta/ xmpmeta"`
	Rdf     Rdf      `xml:"http://www.w3.org/1999/02/22-rdf-syntax-ns# RDF"`
}

type Rdf struct {
	Desc []Node `xml:"http://www.w3.org/1999/02/22-rdf-syntax-ns# Description"`
}

type Node struct {
	XMLName  xml.Name
	Node     []Node `xml:",any"`
	CharData []byte `xml:",chardata"`
}

func Decode(r io.Reader) (*Meta, error) {
	m := new(Meta)
	err := xml.NewDecoder(r).Decode(m)
	if err != nil {
		return nil, err
	}

	return m, nil
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
