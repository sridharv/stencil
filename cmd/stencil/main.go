// Command stencil generates specialized versions of Go packages by replacing types.
//
//Usage
//
// Given a package with an interface "A", stencil can generate a new package with all uses of "A" replaced by "int" (or any other type).
// The generated package is stored in the topmost vendor directory of the repo. If no such directory exists, one is created.
//
// As a trivial example, consider a package "github.com/sridharv/stencil/std/num" with a function Max that computes the maximum value of a list of numbers.
//
//	func Max(v...int) int {
//		// compute max
//	}
//
// This only works for int, but we need a version of Max that works on float32.
// stencil can automatically generate an float32 version by reading an import path with the substitution.
//
// Import the float32 version
//
//  import (
//  	float32_num "github.com/sridharv/stencil/std/num/int/float32"
//  )
//
// and use it in your code
//	func PrintMax(values []float32) {
//		fmt.Println("Max of", values, "=", float32_num.Max(values...))
//	}
//
// This will not compile, since the "github.com/sridharv/stencil/std/num/int/float32" package doesn't exist yet. So in your package directory run
//
//	stencil
//
// This generates a "stencilled" version of the package having int substituted with float32. You can now use it in your code
// If your repo has a vendor directory, this will generate the float32 stencilled version in that vendor directory.
// If not, a vendor directory will be created in your package directory and the stencilled version is generated there.
//
//Supported Types
//
// The set of types that can be replaced are currently restricted to the following:
//
//	* Named types in the stencil
//	* Boolean types
//	* Numeric types
//	* String types
//	* interface{} - Use interface in the import path
//
// These types can be replaced with the same set of types as above, with the exception of interface{}.
//
//With go generate
//
// Add the below line to any package that imports a stencilled package.
//
//	//go:generate stencil
//
// and run
//
//	go generate
//
// in the package directory. You only need one go generate directive per package.
//
//Generate on save
//
// The process of generating stencilled packages can be further streamlined by using stencil as a replacement for goimports.
// Running
// 	stencil -w <path/to/file>
// will also run goimports on your code, while generating any needed stencilled packages.
// You can add this as a separate command to run on save in your editor.
// NOTE: The current version of stencil is slower than goimports and so you may still prefer to use `go generate`.
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
