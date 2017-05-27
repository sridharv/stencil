package main

import (
	"fmt"
	"os"

	"path/filepath"

	"github.com/sridharv/stencil"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s -w [path...]\n", filepath.Base(os.Args[0]))
	os.Exit(1)
}

func main() {
	if len(os.Args) == 1 {
		usage()
	}
	if os.Args[1] != "-w" {
		usage()
	}
	if err := stencil.Process(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		return
	}
}
