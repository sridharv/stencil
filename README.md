# Stencil - Simple code templating for Go

Stencil generates specialized versions of Go packages by replacing types.
This is a prototype of [this proposal](https://www.laddoo.net/p/stencil). It will have bugs and only supports
a subset of the features in the proposal.

Given a package with an interface `A`, stencil can generate a new package with all uses of `A` replaced by `int` (or any other type).
The generated package is stored in the closest vendor directory of the repo. If no such directory exists, one is created.

## Installation

Download stencil using

```
go get -u github.com/sridharv/stencil
```

Install with

```
go install github.com/sridharv/stencil/cmd/stencil
```

## Usage

Detailed documentation is at [![GoDoc](https://godoc.org/github.com/sridharv/stencil/cmd/stencil?status.svg)](https://godoc.org/github.com/sridharv/stencil/cmd/stencil) 

### Example Walkthrough

Install stencil as shown above. Then create a package to act as a stencil. Let's create a generic Sum method.

Create `$GOPATH/src/example/stencil/math/math.go` (or a package of your choice) containing

```
package math

func Sum(n...float64) float64 {
    var r float64
    for _, v := range n {
        r += v
    }
    return r
}
```

Now use it to compute the sum of `int`s by using `stencil` to replace `float64` with `int`.

Create `$GOPATH/src/example/usestencil/main.go` (or another package of your choice) containing

```
package main

//go:generate stencil
import (
    "fmt"
    int_math "example/stencil/math/float64/int"
)

// PrintIntSum prints the sum of all elements of v to stdout.
func main() {
    ints := []int{1, 2, 3, 4, 5}
    fmt.Println("Sum of", v, "=", int_math.Sum(ints...))
}
```

Now run

```
 # Or the path to the main package you created
 cd $GOPATH/src/example/usestencil/ 
```
and then
```
go generate
```

Take a look at `$GOPATH/src/example/usestencil/`. It will have a vendor directory with a specialised version of
`example/stencil/math`. You can now build and run the example with

```
go run main.go
```

## Libraries

A few useful packages that lend themselves to being used with `stencil`.

 * `github.com/sridharv/stencil/std/num` - Max, Min and Sum for numbers. [![GoDoc](https://godoc.org/github.com/sridharv/stencil/std/num?status.svg)](https://godoc.org/github.com/sridharv/stencil/std/num)
 * `github.com/sridharv/stencil/std/slice` - Slice utilities. [![GoDoc](https://godoc.org/github.com/sridharv/stencil/std/slice?status.svg)](https://godoc.org/github.com/sridharv/stencil/std/slice)

## License

Stencil is available under the Apache License. See the LICENSE file for details.

### Contributing

Pull requests are always welcome! Please ensure any changes you send have an accompanying test case.
