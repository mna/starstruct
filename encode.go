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
//   - For slices (including []byte), `starlark:"name,astuple"` to convert to
//     Tuple
//   - For slices (including []byte), `starlark:"name,asset"` to convert to Set
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

		var opts []string
		if rawOpts != "" {
			opts = strings.Split(rawOpts, ",")
		}
		if err := toStarlarkValue(path, nm, fld, dst, opts); err != nil {
			return err
		}
	}
	return nil
}

func toStarlarkValue(path, dstName string, goVal reflect.Value, dst dictGetSetter, opts tagOpt) error {
	key := starlark.String(dstName)
	goTyp := goVal.Type()

	sval, err := convertGoValue(path, goVal, opts)
	if err != nil {
		return err
	}
	if err := dst.SetKey(key, sval); err != nil {
		return fmt.Errorf("failed to set value of key %s with type %s at %s: %w", dstName, goTyp, path, err)
	}
	return nil
}

func convertGoValue(path string, goVal reflect.Value, opts tagOpt) (starlark.Value, error) {
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

	curOpt := opts.current()
	switch {
	case isNil:
		return starlark.None, nil
	case goVal.Kind() == reflect.Bool:
		return starlark.Bool(goVal.Bool()), nil
	case goVal.Kind() == reflect.Float32 || goVal.Kind() == reflect.Float64:
		return starlark.Float(goVal.Float()), nil
	case goVal.Kind() >= reflect.Int && goVal.Kind() <= reflect.Int64:
		return starlark.MakeInt64(goVal.Int()), nil
	case goVal.Kind() >= reflect.Uint && goVal.Kind() <= reflect.Uintptr:
		return starlark.MakeUint64(goVal.Uint()), nil

	case goVal.Kind() == reflect.String:
		if curOpt == "asbytes" {
			return starlark.Bytes(goVal.String()), nil
		}
		return starlark.String(goVal.String()), nil

	case isByteSliceType(goVal.Type()) && curOpt != "aslist" && curOpt != "astuple" && curOpt != "asset":
		if curOpt == "asstring" {
			return starlark.String(goVal.Bytes()), nil
		}
		return starlark.Bytes(goVal.Bytes()), nil

	case goVal.Kind() == reflect.Slice && curOpt != "astuple" && curOpt != "asset":
		n := goVal.Len()
		listVals := make([]starlark.Value, n)
		for i := 0; i < n; i++ {
			v := goVal.Index(i)
			sval, err := convertGoValue(fmt.Sprintf("%s[%d]", path, i), v, opts.shift())
			if err != nil {
				return nil, err
			}
			listVals[i] = sval
		}
		return starlark.NewList(listVals), nil

	case goVal.Kind() == reflect.Slice && curOpt == "astuple":
		n := goVal.Len()
		tupVals := make([]starlark.Value, n)
		for i := 0; i < n; i++ {
			v := goVal.Index(i)
			sval, err := convertGoValue(fmt.Sprintf("%s[%d]", path, i), v, opts.shift())
			if err != nil {
				return nil, err
			}
			tupVals[i] = sval
		}
		return starlark.Tuple(tupVals), nil

	case goVal.Kind() == reflect.Slice && curOpt == "asset":
		n := goVal.Len()
		set := starlark.NewSet(n)
		for i := 0; i < n; i++ {
			v := goVal.Index(i)
			sval, err := convertGoValue(fmt.Sprintf("%s[%d]", path, i), v, opts.shift())
			if err != nil {
				return nil, err
			}
			if err := set.Insert(sval); err != nil {
				return nil, fmt.Errorf("failed to insert value into Set at %s: %w", path, err)
			}
		}
		return set, nil

	case isSetMapType(goVal.Type()):
		n := goVal.Len()
		set := starlark.NewSet(n)
		iter := goVal.MapRange()
		for iter.Next() {
			k, v := iter.Key(), iter.Value()
			if !v.Bool() {
				continue
			}
			sval, err := convertGoValue(fmt.Sprintf("%s[%v]", path, k), k, opts.shift())
			if err != nil {
				return nil, err
			}
			if err := set.Insert(sval); err != nil {
				return nil, fmt.Errorf("failed to insert value into Set at %s: %w", path, err)
			}
		}
		return set, nil

	default:
		return nil, fmt.Errorf("unsupported Go type %s at %s", goTyp, path)
	}
}

func getFieldStruct(path string, strct reflect.Value, dst starlark.Value) error {
	panic("unimplemented")
}

type tagOpt []string

func (t tagOpt) current() string {
	if len(t) > 0 {
		return t[0]
	}
	return ""
}

func (t tagOpt) shift() tagOpt {
	if len(t) <= 1 {
		return tagOpt(nil)
	}
	return tagOpt(t[1:])
}