package starstruct

import "go.starlark.net/starlark"

// ToStarlark stores the values from the Go struct into a destination Starlark
// string dictionary. Existing values in dst, if any, are left untouched if not
// overridden by converting the Go struct.
//
// It supports the following data types from Go to Starlark, and all Go types
// can also be a pointer to that type:
//   - nil (pointer, slice or map) => NoneType
//   - bool => Bool
//   - []byte => Bytes
//   - string => String
//   - float32 or float64 => Float
//   - int, uint, and any sized (u)int => Int
//   - struct => Dict
//   - slice of any supported Go type => List
//   - map[T]bool => Set
//
// Conversion can be further controlled by using struct tags. Besides the
// naming of the starlark variable, a comma-separated argument can be provided
// to control the target encoding. The following arguments are supported:
//   - For string fields, `starlark:"name,asbytes"` to convert to Bytes
//   - For []byte fields, `starlark:"name,asstring"` to convert to String
//   - For []byte ([]uint8) fields, `starlark:"name,aslist"` to convert to List
//     (of Int)
//   - For slices, `starlark:"name,astuple"` to convert to Tuple
//   - For slices, `starlark:"name,asset"` to convert to Set
//
// Any level of conversion arguments can be provided, to support for nested
// conversions, e.g. this would convert to a Set of Tuples of Bytes:
//   - [][]string `starlark:"name,asset,astuple,asbytes"`
//
// It panics if vals is not a struct or a pointer to a struct.
func ToStarlark(vals any, dst starlark.StringDict) error {
	panic("unimplemented")
}
