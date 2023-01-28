package starstruct

import (
	"math"
	"math/big"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

func TestFromStarlark(t *testing.T) {
	type StrctBool struct {
		Bptr  *bool
		B2ptr **bool
		B     bool
	}
	type StrctStr struct {
		S      string
		Sptr   *string
		S2ptr  **string
		Bs     []byte
		BsPtr  *[]byte
		Bs2Ptr **[]byte
	}
	type StrctNums struct {
		I    int
		Iptr *int
		I64  int64 `starlark:"int64"`
		I32  int32 `starlark:"int32"`
		I16  int16 `starlark:"int16"`
		I8   int8  `starlark:"int8"`
		U    uint
		U64  uint64
		U32  uint32
		U16  uint16
		U8   uint8
		Up   uintptr
		F32  float32
		F64  float64
	}
	type StrctDict struct {
		*StrctNums
		StrctStr
		StrctBool `starlark:"bools"`
	}
	type M = map[string]starlark.Value
	dict := func(m M) *starlark.Dict {
		d := starlark.NewDict(len(m))
		for k, v := range m {
			if err := d.SetKey(starlark.String(k), v); err != nil {
				panic(err)
			}
		}
		return d
	}
	truev, falsev := true, false
	tooBig := big.NewInt(1).Add(big.NewInt(1).SetUint64(math.MaxUint64), big.NewInt(1))
	sptr := func(s string) *string { return &s }
	bsptr := func(s string) *[]byte { bs := []byte(s); return &bs }
	iptr := func(i int) *int { return &i }

	cases := []struct {
		name string
		vals map[string]starlark.Value
		dst  any
		want any
		err  string
	}{
		{"nil into *bool lower name", M{"bptr": starlark.None}, &StrctBool{}, StrctBool{Bptr: nil}, ""},
		{"nil into *bool upper name", M{"Bptr": starlark.None}, &StrctBool{}, StrctBool{Bptr: nil}, ""},
		{"nil into non-pointer", M{"I": starlark.None}, &StrctNums{}, nil, `cannot assign None to non-pointer field at I: int`},

		{"true into *bool", M{"bptr": starlark.Bool(true)}, &StrctBool{}, StrctBool{Bptr: &truev}, ""},
		{"true into bool", M{"B": starlark.Bool(true)}, &StrctBool{}, StrctBool{B: true}, ""},
		{"false into bool", M{"b": starlark.Bool(false)}, &StrctBool{}, StrctBool{B: false}, ""},
		{"false into *bool", M{"bptr": starlark.Bool(false)}, &StrctBool{}, StrctBool{Bptr: &falsev}, ""},
		{"true into **bool", M{"b2ptr": starlark.Bool(true)}, &StrctBool{}, nil, `cannot assign Bool to unsupported field type at B2ptr: **bool`},
		{"true into *int", M{"iptr": starlark.Bool(true)}, &StrctNums{}, nil, `cannot assign Bool to unsupported field type at Iptr: *int`},

		{"'a' into string", M{"s": starlark.String("a")}, &StrctStr{}, StrctStr{S: "a"}, ``},
		{"'a' into *string", M{"sptr": starlark.String("a")}, &StrctStr{}, StrctStr{Sptr: sptr("a")}, ``},
		{"'a' into **string", M{"s2ptr": starlark.String("a")}, &StrctStr{}, nil, `cannot assign String to unsupported field type at S2ptr: **string`},
		{"'a' into []byte", M{"bs": starlark.String("a")}, &StrctStr{}, StrctStr{Bs: []byte("a")}, ``},
		{"'a' into *[]byte", M{"bsptr": starlark.String("a")}, &StrctStr{}, StrctStr{BsPtr: bsptr("a")}, ``},
		{"'a' into **[]byte", M{"bs2ptr": starlark.String("a")}, &StrctStr{}, nil, `cannot assign String to unsupported field type at Bs2Ptr: **[]uint8`},

		{"b'abc' into string", M{"s": starlark.Bytes("abc")}, &StrctStr{}, StrctStr{S: "abc"}, ``},
		{"b'abc' into *string", M{"sptr": starlark.Bytes("abc")}, &StrctStr{}, StrctStr{Sptr: sptr("abc")}, ``},
		{"b'abc' into **string", M{"s2ptr": starlark.Bytes("abc")}, &StrctStr{}, nil, `cannot assign Bytes to unsupported field type at S2ptr: **string`},
		{"b'abc' into []byte", M{"bs": starlark.Bytes("abc")}, &StrctStr{}, StrctStr{Bs: []byte("abc")}, ``},
		{"b'abv' into *[]byte", M{"bsptr": starlark.Bytes("abc")}, &StrctStr{}, StrctStr{BsPtr: bsptr("abc")}, ``},
		{"b'abc' into **[]byte", M{"bs2ptr": starlark.Bytes("abc")}, &StrctStr{}, nil, `cannot assign Bytes to unsupported field type at Bs2Ptr: **[]uint8`},

		{"1 into int", M{"i": starlark.MakeInt(1)}, &StrctNums{}, StrctNums{I: 1}, ``},
		{"2 into int8", M{"int8": starlark.MakeInt(2)}, &StrctNums{}, StrctNums{I8: 2}, ``},
		{"3 into int16", M{"int16": starlark.MakeInt(3)}, &StrctNums{}, StrctNums{I16: 3}, ``},
		{"4 into int32", M{"int32": starlark.MakeInt(4)}, &StrctNums{}, StrctNums{I32: 4}, ``},
		{"5 into int64", M{"int64": starlark.MakeInt(5)}, &StrctNums{}, StrctNums{I64: 5}, ``},
		{"1 into uint", M{"U": starlark.MakeUint(1)}, &StrctNums{}, StrctNums{U: 1}, ``},
		{"2 into uint8", M{"U8": starlark.MakeUint(2)}, &StrctNums{}, StrctNums{U8: 2}, ``},
		{"3 into uint16", M{"U16": starlark.MakeUint(3)}, &StrctNums{}, StrctNums{U16: 3}, ``},
		{"4 into uint32", M{"U32": starlark.MakeUint(4)}, &StrctNums{}, StrctNums{U32: 4}, ``},
		{"5 into uint64", M{"U64": starlark.MakeUint(5)}, &StrctNums{}, StrctNums{U64: 5}, ``},
		{"6 into uintptr", M{"Up": starlark.MakeUint(6)}, &StrctNums{}, StrctNums{Up: 6}, ``},
		{"7 into float32", M{"f32": starlark.MakeUint(7)}, &StrctNums{}, StrctNums{F32: 7}, ``},
		{"8 into float64", M{"f64": starlark.MakeUint(8)}, &StrctNums{}, StrctNums{F64: 8}, ``},
		{"-1 into int", M{"i": starlark.MakeInt(-1)}, &StrctNums{}, StrctNums{I: -1}, ``},
		{"-2 into int8", M{"int8": starlark.MakeInt(-2)}, &StrctNums{}, StrctNums{I8: -2}, ``},
		{"-3 into int16", M{"int16": starlark.MakeInt(-3)}, &StrctNums{}, StrctNums{I16: -3}, ``},
		{"-4 into int32", M{"int32": starlark.MakeInt(-4)}, &StrctNums{}, StrctNums{I32: -4}, ``},
		{"-5 into int64", M{"int64": starlark.MakeInt(-5)}, &StrctNums{}, StrctNums{I64: -5}, ``},
		{"-1 into uint", M{"u": starlark.MakeInt(-1)}, &StrctNums{}, nil, `out of range`},
		{"-2 into uint8", M{"u8": starlark.MakeInt(-2)}, &StrctNums{}, nil, `out of range`},
		{"-3 into uint16", M{"u16": starlark.MakeInt(-3)}, &StrctNums{}, nil, `out of range`},
		{"-4 into uint32", M{"u32": starlark.MakeInt(-4)}, &StrctNums{}, nil, `out of range`},
		{"-5 into uint64", M{"u64": starlark.MakeInt(-5)}, &StrctNums{}, nil, `out of range`},
		{"-6 into uintptr", M{"up": starlark.MakeInt(-6)}, &StrctNums{}, nil, `out of range`},
		{"too big into int", M{"i": starlark.MakeUint(math.MaxUint64)}, &StrctNums{}, nil, `out of range`},
		{"too big into int8", M{"int8": starlark.MakeInt(math.MaxInt8 + 1)}, &StrctNums{}, nil, `out of range`},
		{"too big into int16", M{"int16": starlark.MakeInt(math.MaxInt16 + 1)}, &StrctNums{}, nil, `out of range`},
		{"too big into int32", M{"int32": starlark.MakeInt(math.MaxInt32 + 1)}, &StrctNums{}, nil, `out of range`},
		{"too big into int64", M{"int64": starlark.MakeUint(math.MaxInt64 + 1)}, &StrctNums{}, nil, `out of range`},
		{"too big into uint", M{"u": starlark.MakeBigInt(tooBig)}, &StrctNums{}, nil, `out of range`},
		{"too big into uint8", M{"u8": starlark.MakeInt(math.MaxUint8 + 1)}, &StrctNums{}, nil, `out of range`},
		{"too big into uint16", M{"u16": starlark.MakeInt(math.MaxUint16 + 1)}, &StrctNums{}, nil, `out of range`},
		{"too big into uint32", M{"u32": starlark.MakeInt(math.MaxUint32 + 1)}, &StrctNums{}, nil, `out of range`},
		{"too big into uint64", M{"u64": starlark.MakeBigInt(tooBig)}, &StrctNums{}, nil, `out of range`},
		{"too big into uintptr", M{"up": starlark.MakeBigInt(tooBig)}, &StrctNums{}, nil, `out of range`},

		{"1.1 into int", M{"i": starlark.Float(1.1)}, &StrctNums{}, nil, `value cannot be exactly represented`},
		{"1.1 into int8", M{"int8": starlark.Float(1.1)}, &StrctNums{}, nil, `value cannot be exactly represented`},
		{"1.1 into int16", M{"int16": starlark.Float(1.1)}, &StrctNums{}, nil, `value cannot be exactly represented`},
		{"1.1 into int32", M{"int32": starlark.Float(1.1)}, &StrctNums{}, nil, `value cannot be exactly represented`},
		{"1.1 into int64", M{"int64": starlark.Float(1.1)}, &StrctNums{}, nil, `value cannot be exactly represented`},
		{"1.1 into uint", M{"U": starlark.Float(1.1)}, &StrctNums{}, nil, `value cannot be exactly represented`},
		{"1.1 into uint8", M{"U8": starlark.Float(1.1)}, &StrctNums{}, nil, `value cannot be exactly represented`},
		{"1.1 into uint16", M{"U16": starlark.Float(1.1)}, &StrctNums{}, nil, `value cannot be exactly represented`},
		{"1.1 into uint32", M{"U32": starlark.Float(1.1)}, &StrctNums{}, nil, `value cannot be exactly represented`},
		{"1.1 into uint64", M{"U64": starlark.Float(1.1)}, &StrctNums{}, nil, `value cannot be exactly represented`},
		{"1.1 into uintptr", M{"up": starlark.Float(1.1)}, &StrctNums{}, nil, `value cannot be exactly represented`},
		{"1.1 into float32", M{"f32": starlark.Float(1.1)}, &StrctNums{}, StrctNums{F32: 1.1}, ``},
		{"1.1 into float64", M{"f64": starlark.Float(1.1)}, &StrctNums{}, StrctNums{F64: 1.1}, ``},

		{"-1.1 into int", M{"i": starlark.Float(-1.1)}, &StrctNums{}, nil, `value cannot be exactly represented`},
		{"-1.1 into int8", M{"int8": starlark.Float(-1.1)}, &StrctNums{}, nil, `value cannot be exactly represented`},
		{"-1.1 into int16", M{"int16": starlark.Float(-1.1)}, &StrctNums{}, nil, `value cannot be exactly represented`},
		{"-1.1 into int32", M{"int32": starlark.Float(-1.1)}, &StrctNums{}, nil, `value cannot be exactly represented`},
		{"-1.1 into int64", M{"int64": starlark.Float(-1.1)}, &StrctNums{}, nil, `value cannot be exactly represented`},
		{"-1.1 into uint", M{"U": starlark.Float(-1.1)}, &StrctNums{}, nil, `value cannot be exactly represented`},
		{"-1.1 into uint8", M{"U8": starlark.Float(-1.1)}, &StrctNums{}, nil, `value cannot be exactly represented`},
		{"-1.1 into uint16", M{"U16": starlark.Float(-1.1)}, &StrctNums{}, nil, `value cannot be exactly represented`},
		{"-1.1 into uint32", M{"U32": starlark.Float(-1.1)}, &StrctNums{}, nil, `value cannot be exactly represented`},
		{"-1.1 into uint64", M{"U64": starlark.Float(-1.1)}, &StrctNums{}, nil, `value cannot be exactly represented`},
		{"-1.1 into uintptr", M{"Up": starlark.Float(-1.1)}, &StrctNums{}, nil, `value cannot be exactly represented`},
		{"-1.1 into float32", M{"f32": starlark.Float(-1.1)}, &StrctNums{}, StrctNums{F32: -1.1}, ``},
		{"-1.1 into float64", M{"f64": starlark.Float(-1.1)}, &StrctNums{}, StrctNums{F64: -1.1}, ``},

		{"1.0 into int", M{"i": starlark.Float(1.0)}, &StrctNums{}, StrctNums{I: 1}, ``},
		{"1.0 into int8", M{"int8": starlark.Float(1.0)}, &StrctNums{}, StrctNums{I8: 1}, ``},
		{"1.0 into int16", M{"int16": starlark.Float(1.0)}, &StrctNums{}, StrctNums{I16: 1}, ``},
		{"1.0 into int32", M{"int32": starlark.Float(1.0)}, &StrctNums{}, StrctNums{I32: 1}, ``},
		{"1.0 into int64", M{"int64": starlark.Float(1.0)}, &StrctNums{}, StrctNums{I64: 1}, ``},
		{"1.0 into uint", M{"U": starlark.Float(1.0)}, &StrctNums{}, StrctNums{U: 1}, ``},
		{"1.0 into uint8", M{"U8": starlark.Float(1.0)}, &StrctNums{}, StrctNums{U8: 1}, ``},
		{"1.0 into uint16", M{"U16": starlark.Float(1.0)}, &StrctNums{}, StrctNums{U16: 1}, ``},
		{"1.0 into uint32", M{"U32": starlark.Float(1.0)}, &StrctNums{}, StrctNums{U32: 1}, ``},
		{"1.0 into uint64", M{"U64": starlark.Float(1.0)}, &StrctNums{}, StrctNums{U64: 1}, ``},
		{"1.0 into uintptr", M{"Up": starlark.Float(1.0)}, &StrctNums{}, StrctNums{Up: 1}, ``},
		{"1.0 into float32", M{"f32": starlark.Float(1.0)}, &StrctNums{}, StrctNums{F32: 1.0}, ``},
		{"1.0 into float64", M{"f64": starlark.Float(1.0)}, &StrctNums{}, StrctNums{F64: 1.0}, ``},

		{"-1.0 into int", M{"i": starlark.Float(-1.0)}, &StrctNums{}, StrctNums{I: -1}, ``},
		{"-1.0 into int8", M{"int8": starlark.Float(-1.0)}, &StrctNums{}, StrctNums{I8: -1}, ``},
		{"-1.0 into int16", M{"int16": starlark.Float(-1.0)}, &StrctNums{}, StrctNums{I16: -1}, ``},
		{"-1.0 into int32", M{"int32": starlark.Float(-1.0)}, &StrctNums{}, StrctNums{I32: -1}, ``},
		{"-1.0 into int64", M{"int64": starlark.Float(-1.0)}, &StrctNums{}, StrctNums{I64: -1}, ``},
		{"-1.0 into uint", M{"U": starlark.Float(-1.0)}, &StrctNums{}, nil, `out of range`},
		{"-1.0 into uint8", M{"U8": starlark.Float(-1.0)}, &StrctNums{}, nil, `out of range`},
		{"-1.0 into uint16", M{"U16": starlark.Float(-1.0)}, &StrctNums{}, nil, `out of range`},
		{"-1.0 into uint32", M{"U32": starlark.Float(-1.0)}, &StrctNums{}, nil, `out of range`},
		{"-1.0 into uint64", M{"U64": starlark.Float(-1.0)}, &StrctNums{}, nil, `out of range`},
		{"-1.0 into uintptr", M{"Up": starlark.Float(-1.0)}, &StrctNums{}, nil, `out of range`},
		{"-1.0 into float32", M{"f32": starlark.Float(-1.0)}, &StrctNums{}, StrctNums{F32: -1.0}, ``},
		{"-1.0 into float64", M{"f64": starlark.Float(-1.0)}, &StrctNums{}, StrctNums{F64: -1.0}, ``},

		{"too big into int", M{"i": starlark.Float(math.MaxInt + 1)}, &StrctNums{}, nil, `out of range`},
		{"too big into int8", M{"int8": starlark.Float(math.MaxInt8 + 1)}, &StrctNums{}, nil, `out of range`},
		{"too big into int16", M{"int16": starlark.Float(math.MaxInt16 + 1)}, &StrctNums{}, nil, `out of range`},
		{"too big into int32", M{"int32": starlark.Float(math.MaxInt32 + 1)}, &StrctNums{}, nil, `out of range`},
		{"too big into int64", M{"int64": starlark.Float(math.MaxInt64 + 1)}, &StrctNums{}, nil, `out of range`},
		{"too big into uint", M{"U": starlark.Float(math.MaxUint + 1)}, &StrctNums{}, nil, `out of range`},
		{"too big into uint8", M{"U8": starlark.Float(math.MaxUint8 + 1)}, &StrctNums{}, nil, `out of range`},
		{"too big into uint16", M{"U16": starlark.Float(math.MaxUint16 + 1)}, &StrctNums{}, nil, `out of range`},
		{"too big into uint32", M{"U32": starlark.Float(math.MaxUint32 + 1)}, &StrctNums{}, nil, `out of range`},
		{"too big into uintptr", M{"Up": starlark.Float(math.MaxUint64 + 1)}, &StrctNums{}, nil, `out of range`},
		{"too big into float32", M{"f32": starlark.Float(math.MaxFloat64)}, &StrctNums{}, nil, `cannot be exactly represented`},
		{"too big into float64", M{"f64": starlark.Float(math.MaxFloat64 + 1)}, &StrctNums{}, StrctNums{F64: math.MaxFloat64 + 1}, ``},

		{"embedded ptr int", M{"i": starlark.MakeInt(1)}, &StrctDict{}, StrctDict{StrctNums: &StrctNums{I: 1}}, ``},
		{"embedded ptr *int", M{"iptr": starlark.MakeInt(1)}, &StrctDict{}, StrctDict{StrctNums: &StrctNums{Iptr: iptr(1)}}, ``},
		// TODO: it is a bug that StrctNums gets allocated and set, as none of its fields are touched
		{"embedded ptr string", M{"s": starlark.String("abc")}, &StrctDict{}, StrctDict{StrctNums: &StrctNums{}, StrctStr: StrctStr{S: "abc"}}, ``},
		{"embedded ptr *string", M{"sptr": starlark.String("abc")}, &StrctDict{}, StrctDict{StrctNums: &StrctNums{}, StrctStr: StrctStr{Sptr: sptr("abc")}}, ``},
		{"embedded ptr **string", M{"s2ptr": starlark.String("abc")}, &StrctDict{}, nil, `cannot assign String to unsupported field type at StrctStr.S2ptr: **string`},
		{"embedded ptr unprefixed b", M{"B": starlark.Bool(true)}, &StrctDict{}, StrctDict{StrctNums: &StrctNums{}}, ``},
		{"embedded ptr prefixed b", M{"bools": dict(M{"B": starlark.Bool(true)})}, &StrctDict{}, StrctDict{StrctNums: &StrctNums{}, StrctBool: StrctBool{B: true}}, ``},
		{"embedded ptr prefixed *bool", M{"bools": dict(M{"bptr": starlark.Bool(true)})}, &StrctDict{}, StrctDict{StrctNums: &StrctNums{}, StrctBool: StrctBool{Bptr: &truev}}, ``},
		{"embedded ptr prefixed **bool", M{"bools": dict(M{"b2ptr": starlark.Bool(true)})}, &StrctDict{}, nil, `cannot assign Bool to unsupported field type at StrctBool.B2ptr: **bool`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := FromStarlark(c.vals, c.dst)
			if c.err != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), c.err)
				return
			}

			require.NoError(t, err)
			rv := reflect.ValueOf(c.dst)
			require.Equal(t, c.want, rv.Elem().Interface())
		})
	}
}
