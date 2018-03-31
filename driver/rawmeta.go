package driver

import "errors"

type RawMeta struct {
	Name  string
	Bytes []byte
}

var ErrNotReadSeeker = errors.New("metadata: Reader must be a ReadSeeker")
