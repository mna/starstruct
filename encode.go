package starstruct

import (
	"fmt"
	"reflect"
	"strings"

	"go.starlark.net/starlark"
)

// ToStarlark converts the values from the Go struct to corresponding Starlark
// values stored into a destination Starlark string dictionary. Existing values
// in dst, if any, are left untouched unless the Go struct conversion
// overwrites them.
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
// It panics if vals is not a struct or a pointer to a struct. If dst is nil,
// it proceeds with the conversion but the results of it will not be visible to
// the caller (it can be used to validate the Go to Starlark conversion).
func ToStarlark(vals any, dst starlark.StringDict) error {
	strct := reflect.ValueOf(vals)
	oriVal := strct
	for strct.Kind() == reflect.Pointer {
		strct = strct.Elem()
	}
	if strct.Kind() != reflect.Struct {
		panic(fmt.Sprintf("source value is not a struct or a pointer to a struct: %s", oriVal.Type()))
	}
	if dst == nil {
		// results will not be visible to the caller, but it will validate any
		// conversion error.
		dst = make(starlark.StringDict)
	}
	return walkStructEncode("", strct, stringDictValue{dst})
}

func walkStructEncode(path string, strct reflect.Value, dst dictGetSetter) error {
	strctTyp := strct.Type()
	count := strctTyp.NumField()
	for i := 0; i < count; i++ {
		fldTyp := strctTyp.Field(i)
		nm, rawOpts, _ := strings.Cut(fldTyp.Tag.Get("starlark"), ",")
		if !fldTyp.IsExported() || nm == "-" {
			continue
		}

		path := path
		if fldTyp.Name != "" {
			if path != "" {
				path += "."
			}
			path += fldTyp.Name
		}
		fld := strct.Field(i)

		// use the field name as target starlark name, except if the field is an
		// embedded anonymous struct - in this case we will walk this embedded
		// struct as if the fields were in the current struct.
		if nm == "" {
			if fldTyp.Anonymous {
				if err := getFieldStruct(path, fld, dst); err != nil {
					return err
				}
				continue
			}
			nm = fldTyp.Name
		}

		_ = rawOpts
		if err := toStarlarkValue(path, nm, fld, dst); err != nil {
			return err
		}
	}
	return nil
}

func toStarlarkValue(path, dstName string, goVal reflect.Value, dst dictGetSetter) error {
	key := starlark.String(dstName)
	goTyp := goVal.Type()

	var isNil bool
	// allow one level of indirection
	if goTyp.Kind() == reflect.Pointer && goTyp.Elem().Kind() != reflect.Pointer {
		isNil = goVal.IsNil()
		goVal = goVal.Elem()
	}
	// map and slice can also be nil
	if goVal.Kind() == reflect.Map || goVal.Kind() == reflect.Slice {
		isNil = goVal.IsNil()
	}

	switch {
	case isNil:
		if err := dst.SetKey(key, starlark.None); err != nil {
			return fmt.Errorf("failed to set key %s to None at %s: %w", dstName, path, err)
		}

	case goVal.Kind() == reflect.Bool:
		if err := dst.SetKey(key, starlark.Bool(goVal.Bool())); err != nil {
			return fmt.Errorf("failed to set key %s to Bool at %s: %w", dstName, path, err)
		}

	default:
		return fmt.Errorf("unsupported Go type %s at %s", goTyp, path)
	}
	return nil
}

func getFieldStruct(path string, strct reflect.Value, dst starlark.Value) error {
	panic("unimplemented")
}
