[![Go Reference](https://pkg.go.dev/badge/github.com/mna/starstruct.svg)](https://pkg.go.dev/github.com/mna/starstruct)
[![Build Status](https://github.com/mna/starstruct/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/mna/starstruct/actions)

# Starlark to Go struct converter

The `starstruct` package implements conversion of Go structs to [Starlark `StringDict`](https://pkg.go.dev/go.starlark.net/starlark#StringDict) and from Starlark `StringDict` to a Go struct.

It uses the `starlark` struct tag key to indicate the name of the corresponding Starlark `StringDict` key for a field and optional conversion options, as is common in Go marshalers (such as [JSON](https://pkg.go.dev/encoding/json) and [XML](https://pkg.go.dev/encoding/xml)).

Since Starlark is a useful language for configuration, and configuration can often be represented by a well-defined Go struct, the idea for this package is to make it easy to provide the default configuration (as a Go struct) to a Starlark program, and read back the final configuration after the Starlark program's execution, so that the Go code can use the Go configuration struct from there on.

The [code documentation](https://pkg.go.dev/github.com/mna/starstruct) is the canonical source for documentation.

## Installation

Note that `starstruct` requires Go 1.20+.

```
$ go get github.com/mna/starstruct
```

## Example

TODO

## License

The [BSD 3-Clause license](http://opensource.org/licenses/BSD-3-Clause).
