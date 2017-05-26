package main

import (
	"fmt"
	"os"

	"github.com/sridharv/stencil"
)

func main() {
	if err := stencil.Process(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		return
	}
}
