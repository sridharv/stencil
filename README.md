# Stencil - Simple code templating for Go

Stencil generates specialized versions of Go packages by replacing types.

Given a package with an interface `A`, stencil can generate a new package with all uses of `A` replaced by `int` (or any other type).
The generated package is stored in the topmost vendor directory of the repo. If no such directory exists, one is created.

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

Please consult the documentation at [![GoDoc](https://godoc.org/github.com/sridharv/stencil/cmd/stencil?status.svg)](https://godoc.org/github.com/sridharv/stencil/cmd/stencil) 

## License

Lint is available under the Apache License. See the LICENSE file for details.

### Contributing

Pull requests are always welcome! Please ensure any changes you send have an accompanying test case.
