// Package starstruct implements conversion of Go structs to Starlark string
// dictionary (starlark.StringDict) and vice versa. By default, the exported
// struct field's name is used as corresponding Starlark dictionary key. The
// 'starlark' struct tag can be used to alter that behavior and to specify
// other options, as is common in Go marshalers (such as in Go's JSON and XML
// standard library packages).
//
// See the documentation of ToStarlark and FromStarlark for more information
// about the encoding and decoding processing.
//
// The Starlark dictionary can hold any hashable value as a key, but only a
// subset is supported by starstruct. It can only convert to and from
// dictionaries where the key is a valid Go identifier or representable as a
// valid struct tag name, excluding any names and characters with special
// meaning.
//
// Those are:
//   - "-" : the name used to ignore a field
//   - "," : the character used to separate the name and the conversion options
package starstruct
