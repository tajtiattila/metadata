package xmldoc

import ()

type Document struct {
	Root Node
}

type Namespace struct {
	Prefix string
	URI    string
}
