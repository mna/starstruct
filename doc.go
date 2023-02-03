// Package starstruct implements conversion of Go structs to Starlark string
// dictionary (starlark.StringDict) and vice versa. By default, the exported
// struct field's name is used as corresponding Starlark dictionary key. The
// 'starlark' struct tag can be used to alter that behavior and to specify
// other options, as is common in Go marshalers (such as in Go's JSON and XML
// standard library packages). The "-" name can be provided to ignore a struct
// field.
//
// See the documentation of ToStarlark and FromStarlark for more information
// about the encoding and decoding processing.
//
// Because the Starlark dictionary can hold any string as a key, and the Go
// struct tag assigns distinct behaviors to special names such as "-" and uses
// "," to separate the name part from other options, the "#" character can be
// used in the struct tag to indicate that the starlark name is the string that
// follows, up until a matching number of '#' to close it.
//
// Inside those pound characters, the string is treated the same as if it was a
// quoted Go string, with support for the same escape sequences. Note, however,
// that the Go struct tag will decode escape sequences when getting the value
// of the 'starlark' tag, so they need to be encoded with two backslashes.
//
// Here are some examples matching the starlark dictionary key to the struct
// tag that matches it, with an "asbytes" option added to show such options
// would follow:
//   - "-" => `starlark:"#-#,asbytes"`
//   - "*" => `starlark:"#*#,asbytes"`
//   - "," => `starlark:"#,#,asbytes"`
//   - "a##" => `starlark:"###a#####,asbytes"` (3 pounds surrounding the name)
//   - "##"##" => `starlark:"###\\x23#\\\"#####,asbytes"` (3 pounds as leading
//     and trailing, then a doubly-escaped pound (\\x23) so that only 3 pounds are
//     counted as surrounding chars, then the second pound as normal, then an
//     escaped backslash followed by an escaped quote, followed by the last 2 pounds
//     and the trailing 3 pounds!)
package starstruct
