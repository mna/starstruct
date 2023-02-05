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

// CustomConvError wraps an error that occurred in a custom converter with
// additional information about the values and struct path involved.
type CustomConvError struct {
	// Op indicates if this is in a FromStarlark or ToStarlark call.
	Op ConvOp
	// Path indicates the Go struct path to the field in error.
	Path string
	// StarVal is the starlark value in a From conversion, nil otherwise.
	StarVal starlark.Value
	// GoVal is the Go value associated with the error.
	GoVal reflect.Value
	// Err is the error as returned by the custom converter.
	Err error
}

// Unwrap returns the underlying custom converter error.
func (e *CustomConvError) Unwrap() error {
	return e.Err
}

// Error returns the error message for the custom conversion error.
func (e *CustomConvError) Error() string {
	return fmt.Sprintf("%s: custom converter error: %v", e.Path, e.Err)
}

// TypeError represents a starstruct conversion error. Errors returned from
// ToStarlark and FromStarlark may wrap errors of this type - that is, the
// returned error is created by using the Go standard library errors.Join
// function, and the errors wrapped in that call may contain
// starstruct.TypeError errors.
type TypeError struct {
	// Op indicates if this is in a FromStarlark or ToStarlark call.
	Op ConvOp
	// Path indicates the Go struct path to the field in error.
	Path string
	// StarVal is the starlark value in a From conversion, nil otherwise.
	StarVal starlark.Value
	// GoVal is the Go value associated with the error.
	GoVal reflect.Value
	// Embedded is true if the Go value is an embedded struct field.
	Embedded bool
}

// Error returns the error message of the starstruct type error.
func (e *TypeError) Error() string {
	if e.Op == OpFromStarlark {
		if e.StarVal == nil {
			return fmt.Sprintf("%s: cannot convert nil Starlark value to Go type %s", e.Path, e.GoVal.Type())
		}
		return fmt.Sprintf("%s: cannot convert Starlark %s to Go type %s", e.Path, e.StarVal.Type(), e.GoVal.Type())
	}
	if e.Embedded {
		return fmt.Sprintf("%s: unsupported embedded Go type %s", e.Path, e.GoVal.Type())
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
	// Reason indicates the cause of the number conversion failure.
	Reason NumberFailReason
	// Path indicates the Go struct path to the field in error.
	Path string
	// StarNum is the Starlark integer or float value associated with the error.
	StarNum starlark.Value
	// GoVal is the target Go value where the number was attempted to be stored.
	GoVal reflect.Value
}

// Error returns the error message for the number conversion failure.
func (e *NumberError) Error() string {
	if e.Reason == NumCannotExactlyRepresent {
		return fmt.Sprintf("%s: cannot assign Starlark %s to Go type %s: value cannot be exactly represented", e.Path, e.StarNum.Type(), e.GoVal.Type())
	}
	return fmt.Sprintf("%s: cannot assign Starlark %s to Go type %s: value out of range", e.Path, e.StarNum.Type(), e.GoVal.Type())
}

// StarlarkContainerError indicates an error that occurred when inserting a
// value into a Starlark container such as a dictionary or a set. It wraps the
// actual error returned by Starlark and provides additional information about
// where in the Go struct encoding the error occurred.
type StarlarkContainerError struct {
	// Path indicates the Go struct path to the field in error.
	Path string
	// Container is the Starlark Set, Dict or StringDict that failed to insert
	// the value.
	Container starlark.Value
	// Key is the key (always a string) at which the value was being inserted,
	// nil if the container is not a Dict or StringDict.
	Key starlark.Value
	// Value is the value that failed to be inserted in the container.
	Value starlark.Value
	// GoVal is the Go value that converted to the Value being inserted.
	GoVal reflect.Value
	// Err is the error returned by Starlark when inserting into the container.
	// The StarlarkContainerError wraps this error.
	Err error
}

// Unwrap returns the underlying Starlark error.
func (e *StarlarkContainerError) Unwrap() error {
	return e.Err
}

// Error returns the error message for the Starlark container insertion.
func (e *StarlarkContainerError) Error() string {
	if e.Key == nil {
		return fmt.Sprintf("%s: failed to insert Starlark %s into %s: %v", e.Path, e.Value.Type(), e.Container.Type(), e.Err)
	}
	return fmt.Sprintf("%s: failed to insert Starlark %s at key %s into %s: %v", e.Path, e.Value.Type(), e.Key.String(), e.Container.Type(), e.Err)
}
