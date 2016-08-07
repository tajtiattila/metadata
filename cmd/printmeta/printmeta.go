package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/tajtiattila/metadata"
)

func main() {
	flag.Parse()

	for _, fn := range flag.Args() {
		processFile(fn)
	}
}

func processFile(fn string) {
	f, err := os.Open(fn)
	if err != nil {
		log.Println(err)
		return
	}
	defer f.Close()

	m, err := metadata.ParseAt(f)
	if err != nil {
		log.Printf("%s: %s", fn, err)
		return
	}

	fmt.Printf("%s:\n", fn)
	for k, v := range m.Attr {
		fmt.Printf("  %s: %q\n", k, v)
	}
}
