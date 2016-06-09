package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/tajtiattila/exif-go/exif"
	"github.com/tajtiattila/exif-go/exif/exiftag"
)

func main() {
	for _, arg := range os.Args[1:] {
		filepath.Walk(arg, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Println(err)
				return nil
			}
			if info.Mode().IsDir() {
				return nil
			}
			if err := desc(path); err != nil {
				log.Println(err)
			}
			return nil
		})
	}
}

func desc(fn string) error {
	fmt.Println(fn)
	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer f.Close()

	x, err := exif.Decode(f)
	if err != nil {
		return err
	}

	xf := exif.Formatter{x.ByteOrder}
	for _, e := range x.GPS {
		n := exiftag.Id(exiftag.GPS | uint32(e.Tag))
		fmt.Printf("  %s = %s\n", n, xf.Value(e.Type, e.Count, e.Value))
	}

	return nil
}
