package starstruct_test

import (
	"fmt"
	"testing"

	"github.com/mna/starstruct"
	"github.com/stretchr/testify/require"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
)

// This file contains "end-to-end" kinds of tests, where ToStarlark is used to
// convert a Go struct to the Starlark StringDict used as initial globals for a
// Starlark script, and FromStarlark is used to read the script's output back
// into a Go struct and validate the results.

func init() {
	// Otherwise starlark is very restrictive with what you can do to predeclared
	// globals. Set and Recursion are probably not strictly required for those
	// tests.
	resolve.AllowGlobalReassign = true
	resolve.AllowSet = true
	resolve.AllowRecursion = true
}

func TestSingleVar(t *testing.T) {
	// for some reason x += 1 does not work even with AllowGlobalReassign
	const script = `
x = x + 1
`

	globals := make(starlark.StringDict)
	type S struct {
		X int `starlark:"x"`
	}
	in := S{X: 1}
	require.NoError(t, starstruct.ToStarlark(in, globals))

	var th starlark.Thread
	mod, err := starlark.ExecFile(&th, "test", script, globals)
	require.NoError(t, err)
	mergeStringDicts(globals, mod)

	var out S
	require.NoError(t, starstruct.FromStarlark(globals, &out))
	require.Equal(t, S{X: 2}, out)
}

func TestPartialOverride(t *testing.T) {
	const script = `
def set_server(srv):
	srv["ports"].append(2020)

def set_admin(adm):
	adm["name"] = "Nitram"
	adm["Age"] = 44
	adm["bands"] = adm["bands"].union(["Metallica", "Kreator"])
	if adm.get("Created") or adm.get("created") or adm.get("-"):
		found_created = True

set_server(server)
set_admin(admin)
`

	globals := make(starlark.StringDict)
	type Server struct {
		Addr  string `starlark:"addr"`
		Ports []int  `starlark:"ports"`
	}
	type User struct {
		Name    string `starlark:"name"`
		Age     int
		Created bool     `starlark:"-"`
		Bands   []string `starlark:"bands,asset"`
	}
	type S struct {
		Server       Server `starlark:"server"`
		Admin        User   `starlark:"admin"`
		FoundCreated *bool  `starlark:"found_created"`
	}

	in := S{
		Server: Server{Addr: "localhost", Ports: []int{80, 443}},
		Admin:  User{Name: "Martin", Age: 22, Created: true, Bands: []string{"Slayer"}},
	}
	require.NoError(t, starstruct.ToStarlark(in, globals))

	var th starlark.Thread
	mod, err := starlark.ExecFile(&th, "test", script, globals)
	require.NoError(t, err)
	mergeStringDicts(globals, mod)

	var out S
	require.NoError(t, starstruct.FromStarlark(globals, &out))
	require.Equal(t, S{
		Server: Server{
			Addr:  "localhost",
			Ports: []int{80, 443, 2020},
		},
		Admin: User{
			Name:  "Nitram",
			Age:   44,
			Bands: []string{"Slayer", "Metallica", "Kreator"},
		},
		FoundCreated: nil,
	}, out)
}

func TestDeeplyNested(t *testing.T) {
	const script = `
x = {
	"y": {
		"z": {
			"vals": 123,
		}
	}
}
`

	globals := make(starlark.StringDict)

	var th starlark.Thread
	mod, err := starlark.ExecFile(&th, "test", script, globals)
	require.NoError(t, err)
	mergeStringDicts(globals, mod)

	out := struct {
		X struct {
			Y struct {
				Z struct {
					Vals int
				}
			}
		}
	}{}
	want := out
	want.X.Y.Z.Vals = 123
	require.NoError(t, starstruct.FromStarlark(globals, &out))
	require.Equal(t, want, out)
}

func mergeStringDicts(dst starlark.StringDict, vs ...starlark.StringDict) starlark.StringDict {
	if dst == nil {
		dst = make(starlark.StringDict)
	}
	for _, dict := range vs {
		for k, v := range dict {
			dst[k] = v
		}
	}
	return dst
}

// nolint: deadcode,unused
func printGlobals(gs starlark.StringDict) {
	fmt.Println("\nGlobals:")
	for _, name := range gs.Keys() {
		v := gs[name]
		fmt.Printf("%s (%s) = %s\n", name, v.Type(), v.String())
	}
}
