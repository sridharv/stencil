package main

import (
	"fmt"
	"os"

	"path/filepath"

	"flag"

	"github.com/sridharv/stencil"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s -w [path...]\n", filepath.Base(os.Args[0]))
	os.Exit(1)
}

func main() {
	var w bool
	flag.BoolVar(&w, "w", false, "If true, the input files are overwritten after formatting")

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "stencil [-w] [path...]")
		flag.PrintDefaults()
	}
	flag.Parse()

	if err := stencil.Process(flag.Args(), w); err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		return
	}
}
