package starstruct

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"go.starlark.net/starlark"
)

// ToOption is the type of the encoding options that can be provided to the
// ToStarlark function.
type ToOption func(*encoder)

// MaxToErrors sets the maximum numbers of errors to return. If too many errors
// are reached, the error returned by ToStarlark will wrap max + 1 errors, the
// last one being an error indicating that the maximum was reached. If max <=
// 0, all errors will be returned.
func MaxToErrors(max int) ToOption {
	return func(e *encoder) {
		e.maxErrs = max
	}
}

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
// In addition to those conversions, if the Go type is starlark.Value (or a
// pointer to that type), then the starlark value is transferred as-is.
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
// Embedded fields in structs are supported as follows:
//   - The field must be exported
//   - The type of the field must be a struct or a pointer to a struct
//   - If the embedded field has no starlark name specified in its struct tag,
//     the fields of the embedded struct are encoded as if they were part of the
//     parent struct.
//   - If the embedded field has a starlark name specified in its struct tag
//     (and that name is not "-"), the embedded struct is encoded as a starlark
//     dictionary under that name.
//
// ToStarlark panics if vals is not a struct or a pointer to a struct. If dst
// is nil, it proceeds with the conversion but the results of it will not be
// visible to the caller (it can be used to validate the Go to Starlark
// conversion).
func ToStarlark(vals any, dst starlark.StringDict, opts ...ToOption) error {
	strct := reflect.ValueOf(vals)
	oriVal := strct
	for strct.Kind() == reflect.Pointer {
		strct = strct.Elem()
	}
	if strct.Kind() != reflect.Struct {
		if vals == nil {
			panic("source value is not a struct or a pointer to a struct: nil")
		}
		panic(fmt.Sprintf("source value is not a struct or a pointer to a struct: %s", oriVal.Type()))
	}
	if dst == nil {
		// results will not be visible to the caller, but it will validate any
		// conversion error.
		dst = make(starlark.StringDict)
	}

	var e encoder
	for _, opt := range opts {
		opt(&e)
	}
	return e.encode(strct, dst)
}

type encoder struct {
	errs    []error
	maxErrs int
}

func (e *encoder) encode(strct reflect.Value, sdict starlark.StringDict) (err error) {
	defer func() {
		if v := recover(); v != nil {
			if _, ok := v.(tooManyErrs); ok {
				err = errors.Join(e.errs...)
			} else {
				panic(v)
			}
		}
	}()

	e.walkStructEncode("", strct, stringDictValue{sdict})
	err = errors.Join(e.errs...)
	return
}

// TODO: add support for custom encoders, via a func(path, srcVal) (starVal, bool, error)?

func (e *encoder) walkStructEncode(path string, strct reflect.Value, dst dictGetSetter) {
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
				if !isStructOrPtrType(fldTyp.Type) {
					e.recordEmbeddedTypeErr(path, fld)
					continue
				}
				e.walkStructEncode(path, fld, dst)
				continue
			}
			nm = fldTyp.Name
		}

		var opts []string
		if rawOpts != "" {
			opts = strings.Split(rawOpts, ",")
		}
		e.toStarlarkValue(path, nm, fld, dst, opts)
	}
}

func (e *encoder) toStarlarkValue(path, dstName string, goVal reflect.Value, dst dictGetSetter, opts tagOpt) {
	key := starlark.String(dstName)

	sval := e.convertGoValue(path, goVal, opts)
	if err := dst.SetKey(key, sval); err != nil {
		// don't think this error can happen (key is always a string, create set is
		// never immutable)
		e.recordStarContainerErr(path, dst, key, sval, goVal, err)
	}
}

func (e *encoder) convertGoValue(path string, goVal reflect.Value, opts tagOpt) starlark.Value {
	goTyp := goVal.Type()

	var isNil bool
	// allow one level of indirection
	if goTyp.Kind() == reflect.Pointer && goTyp.Elem().Kind() != reflect.Pointer {
		isNil = goVal.IsNil()
		goVal = goVal.Elem()
	}
	// map and slice can also be nil, and starlark.Value interface
	if goVal.Kind() == reflect.Map || goVal.Kind() == reflect.Slice ||
		(goVal.Kind() == reflect.Interface && goVal.Type() == starlarkValueType) {
		isNil = goVal.IsNil()
	}

	curOpt := opts.current()
	switch {
	case isNil:
		return starlark.None
	case goVal.Type() == starlarkValueType:
		return goVal.Interface().(starlark.Value)
	case goVal.Kind() == reflect.Bool:
		return starlark.Bool(goVal.Bool())
	case goVal.Kind() == reflect.Float32 || goVal.Kind() == reflect.Float64:
		return starlark.Float(goVal.Float())
	case goVal.Kind() >= reflect.Int && goVal.Kind() <= reflect.Int64:
		return starlark.MakeInt64(goVal.Int())
	case goVal.Kind() >= reflect.Uint && goVal.Kind() <= reflect.Uintptr:
		return starlark.MakeUint64(goVal.Uint())

	case goVal.Kind() == reflect.String:
		if curOpt == "asbytes" {
			return starlark.Bytes(goVal.String())
		}
		return starlark.String(goVal.String())

	case isByteSliceType(goVal.Type()) && curOpt != "aslist" && curOpt != "astuple" && curOpt != "asset":
		if curOpt == "asstring" {
			return starlark.String(goVal.Bytes())
		}
		return starlark.Bytes(goVal.Bytes())

	case goVal.Kind() == reflect.Slice && curOpt != "astuple" && curOpt != "asset":
		n := goVal.Len()
		listVals := make([]starlark.Value, n)
		for i := 0; i < n; i++ {
			v := goVal.Index(i)
			sval := e.convertGoValue(fmt.Sprintf("%s[%d]", path, i), v, opts.shift())
			listVals[i] = sval
		}
		return starlark.NewList(listVals)

	case goVal.Kind() == reflect.Slice && curOpt == "astuple":
		n := goVal.Len()
		tupVals := make([]starlark.Value, n)
		for i := 0; i < n; i++ {
			v := goVal.Index(i)
			sval := e.convertGoValue(fmt.Sprintf("%s[%d]", path, i), v, opts.shift())
			tupVals[i] = sval
		}
		return starlark.Tuple(tupVals)

	case goVal.Kind() == reflect.Slice && curOpt == "asset":
		n := goVal.Len()
		set := starlark.NewSet(n)
		for i := 0; i < n; i++ {
			v := goVal.Index(i)
			path := fmt.Sprintf("%s[%d]", path, i)
			sval := e.convertGoValue(path, v, opts.shift())
			if err := set.Insert(sval); err != nil {
				e.recordStarContainerErr(path, set, nil, sval, v, err)
			}
		}
		return set

	case isSetMapType(goVal.Type()):
		n := goVal.Len()
		set := starlark.NewSet(n)
		iter := goVal.MapRange()
		for iter.Next() {
			k, v := iter.Key(), iter.Value()
			if !v.Bool() {
				continue
			}
			path := fmt.Sprintf("%s[%v]", path, k)
			sval := e.convertGoValue(path, k, opts.shift())
			if err := set.Insert(sval); err != nil {
				e.recordStarContainerErr(path, set, nil, sval, k, err)
			}
		}
		return set

	case goVal.Kind() == reflect.Struct:
		n := goVal.NumField()
		dict := starlark.NewDict(n)
		e.walkStructEncode(path, goVal, dict)
		return dict

	default:
		e.recordTypeErr(path, goVal)
		// return None to avoid issues with invalid starlark values
		return starlark.None
	}
}

func (e *encoder) recordTypeErr(path string, goVal reflect.Value) {
	err := &TypeError{
		Op:    OpToStarlark,
		Path:  path,
		GoVal: goVal,
	}
	e.recordErr(err)
}

func (e *encoder) recordEmbeddedTypeErr(path string, goVal reflect.Value) {
	err := &TypeError{
		Op:       OpToStarlark,
		Path:     path,
		GoVal:    goVal,
		Embedded: true,
	}
	e.recordErr(err)
}

func (e *encoder) recordStarContainerErr(path string, container, key, val starlark.Value, goVal reflect.Value, starErr error) {
	err := &StarlarkContainerError{
		Path:      path,
		Container: container,
		Key:       key,
		Value:     val,
		GoVal:     goVal,
		Err:       starErr,
	}
	e.recordErr(err)
}

func (e *encoder) recordErr(err error) {
	if e.maxErrs > 0 && len(e.errs) == e.maxErrs {
		e.errs = append(e.errs, errors.New("maximum number of errors reached"))
		panic(tooManyErrs{})
	}
	e.errs = append(e.errs, err)
}

// returns true if t is a struct or pointer to struct.
func isStructOrPtrType(t reflect.Type) bool {
	if t.Kind() == reflect.Struct {
		return true
	}
	return isStructPtrType(t)
}

func isStructPtrType(t reflect.Type) bool {
	return t.Kind() == reflect.Pointer && t.Elem().Kind() == reflect.Struct
}

func isTOrPtrTType(t, T reflect.Type) bool {
	return t == T || (t.Kind() == reflect.Pointer && t.Elem() == T)
}

//func decodeStructTag(tag string) (nm string, opts tagOpt, err error) {
//	if tag == "" {
//		return "", nil, nil
//	}
//	if s := strings.TrimLeft(tag, "#"); len(tag) > len(s) {
//		// the name is encoded in pounds
//		pounds := len(tag) - len(s) // how many pound chars
//		_ = pounds
//	}
//	panic("unimplemented")
//}

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
	return t[1:]
}
