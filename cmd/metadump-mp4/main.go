package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/tajtiattila/metadata/mp4"
)

func main() {
	flag.Parse()

	for _, a := range flag.Args() {
		dumpFile(a)
	}
}

func dumpFile(fn string) error {
	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer f.Close()

	mf, err := mp4.Parse(f)
	if err != nil {
		return err
	}

	showMVHD(mf.Header)
	showBox("", mf.Box)
	return nil
}

func showMVHD(h *mp4.MVHD) {
	if h == nil {
		fmt.Println("MVHD: nil")
		return
	}

	fmt.Println("MVHD: {")
	fmt.Printf("  Version:      %d\n", h.Version)
	fmt.Printf("  Flags:        %x\n", h.Flags)
	fmt.Printf("  DateCreated:  %s\n", h.DateCreated.Format(time.RFC3339))
	fmt.Printf("  DateModified: %s\n", h.DateCreated.Format(time.RFC3339))
	fmt.Printf("  TimeUnit:     %d\n", h.TimeUnit)
	fmt.Printf("  DurationInUnits: %d\n", h.DurationInUnits)
	fmt.Printf("  Extra length: %d\n", len(h.Raw))
	fmt.Println("}")
}

func showBox(pfx string, b mp4.Box) {
	fmt.Printf("%s%q off %d size %d", pfx, b.Type, b.Offset, b.Size)
	if len(b.Child) == 0 {
		fmt.Println()
		return
	}
	fmt.Println(" {")
	for _, c := range b.Child {
		showBox(pfx+"  ", c)
	}
	fmt.Println(pfx + "}")
}
