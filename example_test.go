package starstruct_test

import (
	"fmt"
	"log"

	"github.com/mna/starstruct"
	"go.starlark.net/starlark"
)

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
