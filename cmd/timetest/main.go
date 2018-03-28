package main

import (
	"fmt"
	"os"
	"time"
)

func main() {

	fmt.Printf("Local=%p\n", time.Local)

	t, err := time.ParseInLocation(time.RFC3339, "2018-03-28T10:11:33", time.Local)
	chk(err)
	fmt.Printf("%s %p\n", t, t.Location())

	ts := t.Format(time.RFC3339)
	t2, err := time.ParseInLocation(time.RFC3339, ts, time.Local)
	chk(err)
	fmt.Println(t2.Location())
}

func chk(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
