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

// TypeError represents a starstruct conversion error. Errors returned from
// ToStarlark and FromStarlark may wrap errors of this type - that is, the
// returned error is created by using the Go standard library errors.Join
// function, and the errors wrapped in that call may contain
// starstruct.TypeError errors.
type TypeError struct {
	Op      ConvOp
	Path    string
	StarVal starlark.Value
	GoVal   reflect.Value
}

// Error returns the error message of the starstruct error.
func (e *TypeError) Error() string {
	if e.Op == OpFromStarlark {
		if e.StarVal == nil {
			return fmt.Sprintf("%s: cannot convert nil Starlark value to Go type %s", e.Path, e.GoVal.Type())
		}
		return fmt.Sprintf("%s: cannot convert Starlark %s to Go type %s", e.Path, e.StarVal.Type(), e.GoVal.Type())
	}
	return fmt.Sprintf("%s: unsupported Go type %s", e.Path, e.GoVal.Type())
}

// NumberFailReason defines the failure reasons for a number conversion.
type NumberFailReason byte

// List of number failure reasons.
const (
	NumCannotExactlyRepresent NumberFailReason = iota
	NumOutOfRange
)

// NumberError represents a numeric conversion error from a starlark Int or
// Float to a Go number type. The Reason field indicates why the conversion
// failed: whether it's because the number could not be exactly represented
// in the target Go number type, or because it was out of range.
//
// The distinction is because the source value may be in the range but
// unrepresentable, for example the float 1.234 is in the range of values for
// an int8 Go type, but it cannot be exactly represented - the fraction would
// be lost. Another example is for a float value that requires 64 bits to be
// represented, if stored in a Go float32, it might be inside the range of the
// float32, but not exactly representable (the value would not be the same).
//
// To determine if a value is the same after conversion from float64 to
// float32, starstruct checks if the absolute difference is smaller than
// epsilon.
type NumberError struct {
	Reason  NumberFailReason
	Path    string
	StarNum starlark.Value // always Int or Float
	GoVal   reflect.Value
}

func (e *NumberError) Error() string {
	if e.Reason == NumCannotExactlyRepresent {
		return fmt.Sprintf("%s: cannot assign Starlark %s to Go type %s: value cannot be exactly represented", e.Path, e.StarNum.Type(), e.GoVal.Type())
	}
	return fmt.Sprintf("%s: cannot assign Starlark %s to Go type %s: value out of range", e.Path, e.StarNum.Type(), e.GoVal.Type())
}
