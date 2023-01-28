package starstruct

import (
	"fmt"
	"math"
	"reflect"
	"strings"

	"go.starlark.net/starlark"
)

// FromStarlark loads the starlark values from vals into a destination Go struct.
// It supports the following data types from Starlark to Go:
//   - NoneType => nil (Go field must be a pointer)
//   - Bool     => bool
//   - Bytes    => []byte or string
//   - String   => []byte or string
//   - Float    => float32 or float64
//   - Int      => int, uint, and any sized (u)int if it fits
//   - Dict     => struct
//
// It panics if dst is not a non-nil pointer to an addressable and settable
// struct. If a target field does not exist in the starlark dictionary, it is
// unmodified.
func FromStarlark(vals starlark.StringDict, dst any) error {
	rval := reflect.ValueOf(dst)
	if rval.Kind() != reflect.Pointer {
		panic(fmt.Sprintf("destination value is not a pointer: kind: %s, type: %s", rval.Kind(), rval.Type().Name()))
	}
	if rval.IsNil() {
		panic(fmt.Sprintf("destination value is a nil pointer: kind: %s, type: %s", rval.Kind(), rval.Type().Name()))
	}
	rval = rval.Elem()
	if rval.Kind() != reflect.Struct {
		panic(fmt.Sprintf("destination value is not a pointer to a struct: pointer to kind: %s, type: %s", rval.Kind(), rval.Type().Name()))
	}
	if !rval.CanAddr() || !rval.CanSet() {
		panic(fmt.Sprintf("destination value is a pointer to an unaddressable or unsettable struct: pointer to kind: %s, type: %s", rval.Kind(), rval.Type().Name()))
	}
	return walkStruct("", rval, vals)
}

func walkStruct(path string, strct reflect.Value, vals map[string]starlark.Value) error {
	strctTyp := strct.Type()
	count := strctTyp.NumField()
	for i := 0; i < count; i++ {
		fldTyp := strctTyp.Field(i)
		nm := fldTyp.Tag.Get("starlark")
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

		var tryLower bool
		// use the field name as default lookup value, except if the field is an
		// embedded anonymous struct - in this case we will walk this embedded
		// struct with the current vals.
		if nm == "" {
			if fldTyp.Anonymous {
				if err := setFieldDict(path, fld, vals); err != nil {
					return err
				}
				continue
			}
			nm = fldTyp.Name
			tryLower = true // if no match is found with the field name, try all lowercase
		}

		matchingVal, ok := vals[nm]
		if !ok {
			if tryLower {
				matchingVal, ok = vals[strings.ToLower(nm)]
			}
			if !ok {
				// leave the field unmodified, no matching starlark value
				continue
			}
		}

		switch v := matchingVal.(type) {
		case starlark.NoneType:
			if err := setFieldNone(path, fld); err != nil {
				return err
			}

		case starlark.Bool:
			if err := setFieldBool(path, fld, bool(v)); err != nil {
				return err
			}

		case starlark.Bytes:
			if err := setFieldBytes(path, fld, string(v)); err != nil {
				return err
			}

		case starlark.String:
			if err := setFieldString(path, fld, string(v)); err != nil {
				return err
			}

		case starlark.Int:
			if err := setFieldInt(path, fld, v); err != nil {
				return err
			}

		case starlark.Float:
			if err := setFieldFloat(path, fld, v); err != nil {
				return err
			}

		case *starlark.Dict:
			if err := setFieldDict(path, fld, indexDictItems(v.Items())); err != nil {
				return err
			}

		default:
			if v == nil {
				return fmt.Errorf("nil starlark Value at %s", path)
			}
			return fmt.Errorf("unsupported starlark type %s at %s", v.Type(), path)
		}
	}
	return nil
}

func setFieldNone(path string, fld reflect.Value) error {
	if fld.Kind() != reflect.Pointer {
		return fmt.Errorf("cannot assign None to non-pointer field at %s: %s", path, fld.Type())
	}
	fld.Set(reflect.Zero(fld.Type()))
	return nil
}

func setFieldBool(path string, fld reflect.Value, b bool) error {
	// support a single-level of indirection, in case the value may be None
	if fld.Kind() == reflect.Pointer {
		ptrToTyp := fld.Type().Elem()
		if ptrToTyp.Kind() != reflect.Bool {
			return fmt.Errorf("cannot assign Bool to unsupported field type at %s: %s", path, fld.Type())
		}

		if fld.IsNil() {
			// allocate the *bool value
			fld.Set(reflect.New(ptrToTyp))
		}
		fld = fld.Elem()
	}

	if fld.Kind() != reflect.Bool {
		return fmt.Errorf("cannot assign Bool to non-bool field type at %s: %s", path, fld.Type())
	}
	fld.SetBool(b)
	return nil
}

func setFieldInt(path string, fld reflect.Value, i starlark.Int) error {
	// support a single-level of indirection, in case the value may be None
	if fld.Kind() == reflect.Pointer {
		ptrToTyp := fld.Type().Elem()
		// can be anything between Int and Float64
		if ptrToTyp.Kind() < reflect.Int || ptrToTyp.Kind() > reflect.Float64 {
			return fmt.Errorf("cannot assign Int to unsupported field type at %s: %s", path, fld.Type())
		}

		if fld.IsNil() {
			// allocate the number value
			fld.Set(reflect.New(ptrToTyp))
		}
		fld = fld.Elem()
	}

	if fld.Kind() < reflect.Int || fld.Kind() > reflect.Float64 {
		return fmt.Errorf("cannot assign Int to unsupported field type at %s: %s", path, fld.Type())
	}
	switch fld.Kind() {
	case reflect.Float32, reflect.Float64:
		f, _ := starlark.AsFloat(i)
		fld.SetFloat(f)
	default:
		if err := starlark.AsInt(i, fld.Addr().Interface()); err != nil {
			return fmt.Errorf("cannot assign Int at %s: %w", path, err)
		}
	}
	return nil
}

var (
	epsilon = float64(math.Nextafter32(1, 2) - 1)
)

func setFieldFloat(path string, fld reflect.Value, f starlark.Float) error {
	// support a single-level of indirection, in case the value may be None
	if fld.Kind() == reflect.Pointer {
		ptrToTyp := fld.Type().Elem()
		// can be anything between Int and Float64
		if ptrToTyp.Kind() < reflect.Int || ptrToTyp.Kind() > reflect.Float64 {
			return fmt.Errorf("cannot assign Float to unsupported field type at %s: %s", path, fld.Type())
		}

		if fld.IsNil() {
			// allocate the number value
			fld.Set(reflect.New(ptrToTyp))
		}
		fld = fld.Elem()
	}

	if fld.Kind() < reflect.Int || fld.Kind() > reflect.Float64 {
		return fmt.Errorf("cannot assign Float to unsupported field type at %s: %s", path, fld.Type())
	}

	fv, _ := starlark.AsFloat(f)
	integer, frac := math.Modf(fv)
	switch fld.Kind() {
	case reflect.Float32:
		// NaN and Inf can convert to float32 without issue
		if !math.IsNaN(fv) && !math.IsInf(fv, 0) && math.Abs(float64(float32(fv))-fv) > epsilon {
			return fmt.Errorf("cannot assign Float at %s: value cannot be exactly represented", path)
		}
		fld.SetFloat(fv)

	case reflect.Float64:
		fld.SetFloat(fv)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if math.IsNaN(fv) || math.IsInf(fv, 0) || frac != 0 {
			return fmt.Errorf("cannot assign Float at %s: value cannot be exactly represented", path)
		}

		switch fld.Kind() {
		case reflect.Int:
			if math.Abs(float64(int(integer))-integer) > epsilon {
				return fmt.Errorf("cannot assign Float at %s: value out of range", path)
			}
		case reflect.Int8:
			if math.Abs(float64(int8(integer))-integer) > epsilon {
				return fmt.Errorf("cannot assign Float at %s: value out of range", path)
			}
		case reflect.Int16:
			if math.Abs(float64(int16(integer))-integer) > epsilon {
				return fmt.Errorf("cannot assign Float at %s: value out of range", path)
			}
		case reflect.Int32:
			if math.Abs(float64(int32(integer))-integer) > epsilon {
				return fmt.Errorf("cannot assign Float at %s: value out of range", path)
			}
		case reflect.Int64:
			if math.Abs(float64(int64(integer))-integer) > epsilon {
				return fmt.Errorf("cannot assign Float at %s: value out of range", path)
			}
		}
		fld.SetInt(int64(fv))

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if math.IsNaN(fv) || math.IsInf(fv, 0) || frac != 0 {
			return fmt.Errorf("cannot assign Float at %s: value cannot be exactly represented", path)
		}
		if integer < 0 {
			return fmt.Errorf("cannot assign Float at %s: value out of range", path)
		}

		switch fld.Kind() {
		case reflect.Uint:
			if math.Abs(float64(uint(integer))-integer) > epsilon {
				return fmt.Errorf("cannot assign Float at %s: value out of range", path)
			}
		case reflect.Uintptr:
			if math.Abs(float64(uintptr(integer))-integer) > epsilon {
				return fmt.Errorf("cannot assign Float at %s: value out of range", path)
			}
		case reflect.Uint8:
			if math.Abs(float64(uint8(integer))-integer) > epsilon {
				return fmt.Errorf("cannot assign Float at %s: value out of range", path)
			}
		case reflect.Uint16:
			if math.Abs(float64(uint16(integer))-integer) > epsilon {
				return fmt.Errorf("cannot assign Float at %s: value out of range", path)
			}
		case reflect.Uint32:
			if math.Abs(float64(uint32(integer))-integer) > epsilon {
				return fmt.Errorf("cannot assign Float at %s: value out of range", path)
			}
		}
		fld.SetUint(uint64(integer))
	}
	return nil
}

func setFieldDict(path string, fld reflect.Value, dict map[string]starlark.Value) error {
	// support a single-level of indirection, in case the value may be None
	if fld.Kind() == reflect.Pointer {
		ptrToTyp := fld.Type().Elem()
		// must be a struct
		if ptrToTyp.Kind() != reflect.Struct {
			return fmt.Errorf("cannot assign Dict to unsupported field type at %s: %s", path, fld.Type())
		}

		if fld.IsNil() {
			// allocate the struct value
			fld.Set(reflect.New(ptrToTyp))
		}
		fld = fld.Elem()
	}

	if fld.Kind() != reflect.Struct {
		return fmt.Errorf("cannot assign Dict to unsupported field type at %s: %s", path, fld.Type())
	}
	return walkStruct(path, fld, dict)
}

func setFieldBytes(path string, fld reflect.Value, s string) error {
	return setFieldBytesOrString(path, "Bytes", fld, s)
}

func setFieldString(path string, fld reflect.Value, s string) error {
	return setFieldBytesOrString(path, "String", fld, s)
}

func setFieldBytesOrString(path, label string, fld reflect.Value, s string) error {
	byteSlice := isByteSliceType(fld.Type())

	// support a single-level of indirection, in case the value may be None
	if fld.Kind() == reflect.Pointer {
		ptrToTyp := fld.Type().Elem()
		byteSlice = isByteSliceType(ptrToTyp)
		if ptrToTyp.Kind() != reflect.String && !byteSlice {
			return fmt.Errorf("cannot assign %s to unsupported field type at %s: %s", label, path, fld.Type())
		}

		if fld.IsNil() {
			// allocate the *string or *[]byte value
			fld.Set(reflect.New(ptrToTyp))
		}
		fld = fld.Elem()
	}

	if fld.Kind() != reflect.String && !byteSlice {
		return fmt.Errorf("cannot assign %s to unsupported field type at %s: %s", label, path, fld.Type())
	}
	if byteSlice {
		fld.SetBytes([]byte(s))
		return nil
	}
	fld.SetString(s)
	return nil
}

func isByteSliceType(t reflect.Type) bool {
	if t.Kind() != reflect.Slice {
		return false
	}
	et := t.Elem()
	return et.Kind() == reflect.Uint8
}

func indexDictItems(keyVals []starlark.Tuple) map[string]starlark.Value {
	m := make(map[string]starlark.Value, len(keyVals))
	for _, kv := range keyVals {
		k, _ := starlark.AsString(kv[0])
		m[k] = kv[1]
	}
	return m
}
