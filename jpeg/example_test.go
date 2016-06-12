package jpeg_test

import (
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

	for scanner.NextChunk() {
		const app1 = 0xe1
		if scanner.IsChunk(app1, []byte("Exif\x00\x00")) {
			_, p, err := scanner.ReadChunk()
			if err != nil {
				break
			}

			// do something with exif
			fmt.Printf("% .32x", p)
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
		const app1 = 0xe1
		if scanner.IsChunk(app1, []byte("Exif\x00\x00")) {
			// read the Exif
			_, exif, err := scanner.ReadChunk()
			if err != nil {
				break
			}

			// do something with the Exif

			// write new Exif
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
