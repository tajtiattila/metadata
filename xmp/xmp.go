package xmp

import (
	"encoding/xml"
	"errors"
	"io"

	"github.com/tajtiattila/xmlutil"
)

type Meta struct {
	Doc xmlutil.Document

	rdf *xmlutil.Node

	// attr maps xmp attribute names to their respective nodes
	attr map[xml.Name]*xmlutil.Node

	// descr maps xml namespaces such as xmpns, exifns...
	// to their respecrive <rdf:Description> nodes.
	descr map[string]*xmlutil.Node
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

	m.rdf = root

	m.attr = make(map[xml.Name]*xmlutil.Node)
	for _, n := range root.Child {
		m.cacheNodes(n)
	}

	return nil
}

func (m *Meta) cacheNodes(n *xmlutil.Node) {
	if n.Name != xmlName(rdfns, "Description") {
		return
	}

	for _, a := range n.Attr {
		if a.Name.Space == "xmlns" {
			if m.descr == nil {
				m.descr = make(map[string]*xmlutil.Node)
			}
			m.descr[a.Value] = n
		}
	}

	for _, c := range n.Child {
		m.attr[c.Name] = c
	}
}

func (m *Meta) getDescr(ns string) *xmlutil.Node {
	n, ok := m.descr[ns]
	if ok {
		return n
	}

	if m.rdf == nil {
		m.rdf = &xmlutil.Node{
			Name: xmlName(rdfns, "RDF"),
			Attr: xattr("xmlns", "rdf", rdfns),
		}
		m.Doc.Node = m.rdf
	}

	a := []string{rdfns, "about", ""}

	if prefix, ok := namespacePrefixMap[ns]; ok {
		a = append(a, "xmlns", prefix, ns)
	}

	descr := &xmlutil.Node{
		Name: xmlName(rdfns, "Description"),
		Attr: xattr(a...),
	}
	m.rdf.Child = append(m.rdf.Child, descr)

	if m.descr == nil {
		m.descr = make(map[string]*xmlutil.Node)
	}
	m.descr[ns] = descr

	return descr
}

func xattr(v ...string) []xml.Attr {
	if len(v)%3 != 0 {
		panic("invalid attr triplet")
	}
	var attrs []xml.Attr
	for i := 0; i < len(v); i += 3 {
		attrs = append(attrs, xml.Attr{
			Name: xml.Name{
				Space: v[i],
				Local: v[i+1],
			},
			Value: v[i+2],
		})
	}
	return attrs
}

var namespacePrefixMap = map[string]string{
	xmpns:  "xmp",
	exifns: "exif",
	tiffns: "tiff",

	"http://cipa.jp/exif/1.0/": "exifEX",
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
