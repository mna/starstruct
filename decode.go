package starstruct

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"

	"go.starlark.net/starlark"
)

// FromStarlark loads the starlark values from vals into a destination Go
// struct. It supports the following data types from Starlark to Go, and all Go
// types can also be a pointer to that type:
//   - NoneType => nil (Go field must be a pointer, slice or map)
//   - Bool     => bool
//   - Bytes    => []byte or string
//   - String   => []byte or string
//   - Float    => float32 or float64
//   - Int      => int, uint, and any sized (u)int if it fits
//   - Dict     => struct
//   - List     => slice of any supported Go type
//   - Tuple    => slice of any supported Go type
//   - Set      => map[T]bool or []T where T is any supported Go type
//
// In addition to those conversions, if the Go type is starlark.Value (or a
// pointer to that type), then the starlark value is assigned as-is.
//
// It panics if dst is not a non-nil pointer to an addressable and settable
// struct. If a target field does not exist in the starlark dictionary, it is
// unmodified.
//
// Decoding into a slice follows the same behavior as JSON umarshaling: it
// resets the slice length to zero and then appends each element to the slice.
// As a special case, to decode an empty starlark List, Tuple or Set into a
// slice, it replaces the slice with a new empty slice.
//
// Decoding a Set into a map also follows the same behavior as JSON
// unmarshaling: if the map is nil, it allocates a new map. Otherwise it reuses
// the existing map, keeping existing entries. It then stores each Set key with
// a true value into the map.
//
// Embedded fields in structs are supported as follows:
//   - The field must be exported
//   - The type of the field must be a struct or a pointer to a struct
//   - If the embedded field has no starlark name specified in its struct tag,
//     the starlark values are decoded into the fields of the embedded struct as
//     if they were part of the parent struct.
//   - If the embedded field has a starlark name specified in its struct tag
//     (and that name is not "-"), the starlark dictionary corresponding to that
//     name is decoded to that embedded struct.
func FromStarlark(vals starlark.StringDict, dst any) error {
	if dst == nil {
		panic("destination value is not a pointer to a struct: nil")
	}

	rval := reflect.ValueOf(dst)
	if !isStructPtrType(rval.Type()) {
		panic(fmt.Sprintf("destination value is not a pointer to a struct: %s", rval.Type()))
	}
	if rval.IsNil() {
		panic(fmt.Sprintf("destination value is a nil pointer: %s", rval.Type()))
	}

	oriVal := rval
	rval = rval.Elem()
	if !rval.CanAddr() || !rval.CanSet() {
		panic(fmt.Sprintf("destination value is a pointer to an unaddressable or unsettable struct: %s", oriVal.Type()))
	}

	var d decoder
	return d.decode(rval, vals)
}

type decoder struct {
	errs    []error
	maxErrs int
	decoded map[dictGetSetter]map[string]bool
	restDst map[dictGetSetter]reflect.Value
}

func (d *decoder) decode(strct reflect.Value, sdict starlark.StringDict) (err error) {
	defer func() {
		if e := recover(); e != nil {
			if _, ok := e.(tooManyErrs); ok {
				err = errors.Join(d.errs...)
			}
		}
	}()

	d.walkStructDecode("", strct, stringDictValue{sdict})
	err = errors.Join(d.errs...)
	return
}

// TODO: maybe add support for a "rest" map[string]starlark.Value for
// dictionary values that were not decoded to fields?
// TODO: add support for custom decoders, via a func(path, starVal, dstVal) (bool, error)?

func (d *decoder) walkStructDecode(path string, strct reflect.Value, vals dictGetSetter) (didSet bool) {
	strctTyp := strct.Type()
	count := strctTyp.NumField()
	for i := 0; i < count; i++ {
		fldTyp := strctTyp.Field(i)
		nm, _, _ := strings.Cut(fldTyp.Tag.Get("starlark"), ",")
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
				ok, err := d.setFieldDict(path, fld, vals)
				if err != nil {
					return didSet, err
				}
				if ok {
					didSet = true
				}
				continue
			}
			nm = fldTyp.Name
			tryLower = true // if no match is found with the field name, try all lowercase
		}

		matchingVal, ok, _ := vals.Get(starlark.String(nm)) // cannot fail, key is a string
		if !ok {
			if tryLower {
				matchingVal, ok, _ = vals.Get(starlark.String(strings.ToLower(nm)))
			}
			if !ok {
				// leave the field unmodified, no matching starlark value
				continue
			}
		}

		// at this point, the struct field has a matching starlark value, so it
		// will either set it or return an error.
		didSet = true
		if err := d.fromStarlarkValue(path, matchingVal, fld); err != nil {
			return didSet, err
		}
	}
	return didSet, nil
}

func (d *decoder) fromStarlarkValue(path string, starVal starlark.Value, dst reflect.Value) {
	// if destination is starlark.Value interface (or a pointer to it), assign
	// it directly, as-is.
	if t := dst.Type(); isTOrPtrTType(t, starlarkValueType) {
		if err := d.setFieldStarlark(path, dst, starVal); err != nil {
			return err
		}
		return nil
	}

	switch v := starVal.(type) {
	case starlark.NoneType:
		if err := d.setFieldNone(path, dst); err != nil {
			return err
		}

	case starlark.Bool:
		if err := d.setFieldBool(path, dst, bool(v)); err != nil {
			return err
		}

	case starlark.Bytes:
		if err := d.setFieldBytes(path, dst, v); err != nil {
			return err
		}

	case starlark.String:
		if err := d.setFieldString(path, dst, v); err != nil {
			return err
		}

	case starlark.Int:
		if err := d.setFieldInt(path, dst, v); err != nil {
			return err
		}

	case starlark.Float:
		if err := d.setFieldFloat(path, dst, v); err != nil {
			return err
		}

	case *starlark.Dict:
		if _, err := d.setFieldDict(path, dst, v); err != nil {
			return err
		}

	case *starlark.List:
		if err := d.setFieldList(path, dst, v); err != nil {
			return err
		}

	case starlark.Tuple:
		if err := d.setFieldTuple(path, dst, v); err != nil {
			return err
		}

	case *starlark.Set:
		if err := d.setFieldSet(path, dst, v); err != nil {
			return err
		}

	default:
		if v == nil {
			return fmt.Errorf("nil starlark Value at %s", path)
		}
		return fmt.Errorf("unsupported starlark type %s (%T) at %s", v.Type(), v, path)
	}
	return nil
}

func (d *decoder) setFieldNone(path string, fld reflect.Value) error {
	if fld.Kind() != reflect.Pointer && fld.Kind() != reflect.Slice && fld.Kind() != reflect.Map {
		return fmt.Errorf("cannot assign None to unsupported field type at %s: %s", path, fld.Type())
	}
	fld.Set(reflect.Zero(fld.Type()))
	return nil
}

func (d *decoder) setFieldStarlark(path string, fld reflect.Value, v starlark.Value) error {
	// support a single-level of indirection for consistency with other types
	if fld.Kind() == reflect.Pointer {
		ptrToTyp := fld.Type().Elem()
		if ptrToTyp != starlarkValueType {
			return fmt.Errorf("cannot assign starlark.Value to unsupported field type at %s: %s", path, fld.Type())
		}

		if fld.IsNil() {
			// allocate the *starlark.Value value
			fld.Set(reflect.New(ptrToTyp))
		}
		fld = fld.Elem()
	}

	if fld.Type() != starlarkValueType {
		return fmt.Errorf("cannot assign starlark.Value to unsupported field type at %s: %s", path, fld.Type())
	}
	fld.Set(reflect.ValueOf(v))
	return nil
}

func (d *decoder) setFieldBool(path string, fld reflect.Value, b bool) error {
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
		return fmt.Errorf("cannot assign Bool to unsupported field type at %s: %s", path, fld.Type())
	}
	fld.SetBool(b)
	return nil
}

func (d *decoder) setFieldInt(path string, fld reflect.Value, i starlark.Int) error {
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

var epsilon = float64(math.Nextafter32(1, 2) - 1)

func (d *decoder) setFieldFloat(path string, fld reflect.Value, f starlark.Float) error {
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

func (d *decoder) setFieldDict(path string, fld reflect.Value, dict dictGetSetter) (didSet bool, err error) {
	var ptrToStrct reflect.Value

	// support a single-level of indirection, in case the value may be None
	if fld.Kind() == reflect.Pointer {
		ptrToStrct = fld
		ptrToTyp := fld.Type().Elem()
		// must be a struct
		if ptrToTyp.Kind() != reflect.Struct {
			return didSet, fmt.Errorf("cannot assign Dict to unsupported field type at %s: %s", path, fld.Type())
		}

		if fld.IsNil() {
			// allocate the struct value, but do not set it yet on the pointer, will
			// only set it if something was set on the struct.
			fld = reflect.New(ptrToTyp)
		}
		fld = fld.Elem()
	}

	if fld.Kind() != reflect.Struct {
		return didSet, fmt.Errorf("cannot assign Dict to unsupported field type at %s: %s", path, fld.Type())
	}
	didSet, err = d.walkStructDecode(path, fld, dict)
	if didSet && ptrToStrct.Kind() == reflect.Pointer {
		ptrToStrct.Set(fld.Addr())
	}
	return didSet, err
}

func (d *decoder) setFieldList(path string, fld reflect.Value, list *starlark.List) error {
	return d.setFieldIterator(path, "List", fld, list)
}

func (d *decoder) setFieldTuple(path string, fld reflect.Value, tup starlark.Tuple) error {
	return d.setFieldIterator(path, "Tuple", fld, tup)
}

type iterable interface {
	starlark.Iterable
	Len() int
}

func (d *decoder) setFieldIterator(path, label string, fld reflect.Value, iter iterable) error {
	// support a single-level of indirection, in case the value may be None (even
	// though it wouldn't be necessary as slice can be nil, but for consistency
	// with other types)
	if fld.Kind() == reflect.Pointer {
		ptrToTyp := fld.Type().Elem()

		// must be a slice
		if ptrToTyp.Kind() != reflect.Slice {
			return fmt.Errorf("cannot assign %s to unsupported field type at %s: %s", label, path, fld.Type())
		}

		if fld.IsNil() {
			// allocate the pointer to slice value
			fld.Set(reflect.New(ptrToTyp))
		}
		fld = fld.Elem()
	}

	if fld.Kind() != reflect.Slice {
		return fmt.Errorf("cannot assign %s to unsupported field type at %s: %s", label, path, fld.Type())
	}
	elemTyp := fld.Type().Elem()

	count := iter.Len()
	if count == 0 {
		// special-case to behave the same as JSON Unmarshal: to unmarshal an empty
		// JSON array into a slice, Unmarshal replaces the slice with a new empty
		// slice.
		fld.Set(reflect.MakeSlice(reflect.SliceOf(elemTyp), 0, 0))
		return nil
	}

	if count > fld.Cap() {
		// replace the slice with one that has the sufficient capacity
		fld.Set(reflect.MakeSlice(reflect.SliceOf(elemTyp), 0, count))
	} else {
		fld.SetLen(0)
	}

	it := iter.Iterate()
	defer it.Done()
	var newVal starlark.Value
	var i int
	for it.Next(&newVal) {
		newElem := reflect.New(elemTyp).Elem()
		if err := d.fromStarlarkValue(fmt.Sprintf("%s[%d]", path, i), newVal, newElem); err != nil {
			return err
		}
		fld.Set(reflect.Append(fld, newElem))
		i++
	}
	return nil
}

var trueValue = reflect.ValueOf(true)

func (d *decoder) setFieldSet(path string, fld reflect.Value, set *starlark.Set) error {
	if fldTyp := fld.Type(); fldTyp.Kind() == reflect.Slice || fldTyp.Kind() == reflect.Pointer && fldTyp.Elem().Kind() == reflect.Slice {
		// same as decoding a List/Tuple
		return d.setFieldIterator(path, "Set", fld, set)
	}

	// support a single-level of indirection, in case the value may be None (even
	// though it wouldn't be necessary as map can be nil, but for consistency
	// with other types)
	if fld.Kind() == reflect.Pointer {
		ptrToTyp := fld.Type().Elem()

		if !isSetMapType(ptrToTyp) {
			return fmt.Errorf("cannot assign Set to unsupported field type at %s: %s", path, fld.Type())
		}

		if fld.IsNil() {
			// allocate the pointer to map value
			fld.Set(reflect.New(ptrToTyp))
		}
		fld = fld.Elem()
	}

	if !isSetMapType(fld.Type()) {
		return fmt.Errorf("cannot assign Set to unsupported field type at %s: %s", path, fld.Type())
	}
	keyTyp, elemTyp := fld.Type().Key(), fld.Type().Elem()

	count := set.Len()

	// mimic the JSON unmarshal behaviour: if the map is nil, allocate one,
	// otherwise the existing map is reused, with the set elements being added to
	// the map.
	if fld.IsNil() {
		mapTyp := reflect.MapOf(keyTyp, elemTyp)
		fld.Set(reflect.MakeMapWithSize(mapTyp, count))
	}

	it := set.Iterate()
	defer it.Done()
	var newVal starlark.Value
	var i int
	for it.Next(&newVal) {
		newKey := reflect.New(keyTyp).Elem()
		if err := d.fromStarlarkValue(fmt.Sprintf("%s[%d]", path, i), newVal, newKey); err != nil {
			return err
		}
		fld.SetMapIndex(newKey, trueValue)
		i++
	}
	return nil
}

func (d *decoder) setFieldBytes(path string, fld reflect.Value, s starlark.Bytes) {
	d.setFieldBytesOrString(path, fld, s, string(s))
}

func (d *decoder) setFieldString(path string, fld reflect.Value, s starlark.String) {
	d.setFieldBytesOrString(path, fld, s, string(s))
}

func (d *decoder) setFieldBytesOrString(path string, fld reflect.Value, v starlark.Value, s string) {
	byteSlice := isByteSliceType(fld.Type())

	// support a single-level of indirection, in case the value may be None
	if fld.Kind() == reflect.Pointer {
		ptrToTyp := fld.Type().Elem()
		byteSlice = isByteSliceType(ptrToTyp)
		if ptrToTyp.Kind() != reflect.String && !byteSlice {
			d.recordErr(path, v, fld)
			return
		}

		if fld.IsNil() {
			// allocate the *string or *[]byte value
			fld.Set(reflect.New(ptrToTyp))
		}
		fld = fld.Elem()
	}

	if fld.Kind() != reflect.String && !byteSlice {
		d.recordErr(path, v, fld)
		return
	}
	if byteSlice {
		fld.SetBytes([]byte(s))
	} else {
		fld.SetString(s)
	}
}

// sentinel type for the panic raised when the maximum number of errors is
// reached.
type tooManyErrs struct{}

func (d *decoder) recordErr(path string, starVal starlark.Value, goVal reflect.Value) {
	err := &Error{
		Op:      OpFromStarlark,
		Path:    path,
		StarVal: starVal,
		GoVal:   goVal,
	}

	d.errs = append(d.errs, err)
	if d.maxErrs > 0 && len(d.errs) >= d.maxErrs {
		d.errs = append(d.errs, errors.New("maximum number of errors reached"))
		panic(tooManyErrs{})
	}
}

// nolint: unused
type ignore struct{ x starlark.Value }

var starlarkValueType = reflect.TypeOf(ignore{}).Field(0).Type

func isByteSliceType(t reflect.Type) bool {
	if t.Kind() != reflect.Slice {
		return false
	}
	return t.Elem().Kind() == reflect.Uint8
}

func isSetMapType(t reflect.Type) bool {
	if t.Kind() != reflect.Map {
		return false
	}
	return t.Elem().Kind() == reflect.Bool
}
