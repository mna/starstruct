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

From `example_test.go`:

```golang
func Example() {
	const script = `
def set_server(srv):
	srv["ports"].append(2020)

def set_admin(adm):
	adm["name"] = "root"
	adm["is_admin"] = True
	roles = adm["roles"]
	adm["roles"] = roles.union(["editor", "admin"])

set_server(server)
set_admin(user)
`

	type Server struct {
		Addr  string `starlark:"addr"`
		Ports []int  `starlark:"ports"`
	}
	type User struct {
		Name    string   `starlark:"name"`
		IsAdmin bool     `starlark:"is_admin"`
		Ignored int      `starlark:"-"`
		Roles   []string `starlark:"roles,asset"`
	}
	type S struct {
		Server Server `starlark:"server"`
		User   User   `starlark:"user"`
	}

	// initialize with default values for the starlark script
	s := S{
		Server: Server{Addr: "localhost", Ports: []int{80, 443}},
		User:   User{Name: "Martin", Roles: []string{"viewer"}, Ignored: 42},
	}
	initialVars := make(starlark.StringDict)
	if err := starstruct.ToStarlark(s, initialVars); err != nil {
		log.Fatal(err)
	}

	// execute the starlark script (it doesn't create any new variables, but if
	// it did, we would capture them in outputVars and merge all global vars
	// together before calling FromStarlark).
	var th starlark.Thread
	outputVars, err := starlark.ExecFile(&th, "example", script, initialVars)
	if err != nil {
		log.Fatal(err)
	}
	allVars := mergeStringDicts(nil, initialVars, outputVars)

	if err := starstruct.FromStarlark(allVars, &s); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%v", s)

	// Output:
	// {{localhost [80 443 2020]} {root true 42 [viewer editor admin]}}
}
```

## License

The [BSD 3-Clause license](http://opensource.org/licenses/BSD-3-Clause).
