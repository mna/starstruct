package starstruct

import (
	"fmt"
	"reflect"

	"go.starlark.net/starlark"
)

// ConvOp is the type indicating the conversion operation, either ToStarlark
// or FromStarlark.
type ConvOp string

// List of ConvOp conversion operations.
const (
	OpToStarlark   ConvOp = "to"
	OpFromStarlark ConvOp = "from"
)

// Error represents a starstruct conversion error. Errors returned from
// ToStarlark and FromStarlark wrap errors of this type - that is, the returned
// error is created by using the Go standard library errors.Join function, and
// the errors wrapped in that call are starstruct.Error errors, except from a
// possible basic string error if there were too many errors.
type Error struct {
	Op      ConvOp
	Path    string
	StarVal starlark.Value
	GoVal   reflect.Value
}

// Error returns the error message of the starstruct error.
func (e *Error) Error() string {
	if e.Op == OpFromStarlark {
		return fmt.Sprintf("%s: cannot convert Starlark %s to Go type %s", e.Path, e.StarVal.Type(), e.GoVal.Type())
	}
	return fmt.Sprintf("%s: unsupported Go type %s", e.Path, e.GoVal.Type())
}
