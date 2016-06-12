package jpeg_test

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/tajtiattila/metadata/jpeg"
)

// Use Scanner to find the Exif in a JPEG file.
func ExampleScanner() {
	scanner, err := jpeg.NewScanner(os.Stdin)
	if err != nil {
		fmt.Printf("jpeg error: %v", err)
		return
	}

	for scanner.Next() {
		if !scanner.StartChunk() {
			continue
		}

		p := scanner.Bytes()
		if len(p) > 4 && bytes.HasPrefix(p[4:], []byte("Exif\x00\x00")) {
			p, err = scanner.ReadChunk()
			if err != nil {
				break
			}

			// do something with exif
			fmt.Printf("% .2x", p[4:])
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("jpeg error: %v", err)
	}
}

func ExampleScanner_copy() {
	input, output := os.Stdin, os.Stdout

	scanner, err := jpeg.NewScanner(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "jpeg error: %v", err)
		return
	}

	var werr error
	for werr == nil && scanner.Next() {
		p, err := scanner.ReadChunk()
		if err != nil {
			break
		}

		const app1 = 0xe1
		if len(p) > 4 && p[0] == 0xff && p[1] == app1 &&
			bytes.HasPrefix(p[4:], []byte("Exif\x00\x00")) {

			exif := p[4:]

			// do something with exif

			// write new exif
			werr = jpeg.WriteChunk(output, app1, exif)
		} else {
			// copy other data from source
			var n int
			n, werr = output.Write(scanner.Bytes())
			if n != scanner.Len() {
				werr = io.ErrShortWrite
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "jpeg error: %v", err)
	}

	if werr != nil {
		fmt.Fprintf(os.Stderr, "write error: %v", werr)
	}
}
