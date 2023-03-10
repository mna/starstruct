package starstruct

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

func TestFromStarlark(t *testing.T) {
	type StrctBool struct {
		Bptr    *bool
		B2ptr   **bool
		B       bool
		Ignored **string `starlark:"-"`
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
		I64  int64 `starlark:"int64,someopt,otheropt"`
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
		Fptr *float64
	}

	type StrctDict struct {
		*StrctNums
		StrctStr
		StrctBool `starlark:"bools"`
	}

	type StrctList struct {
		I        []int
		S        []string
		Sptr     []*string
		PtrIptr  *[]*int
		Strct    []StrctBool
		StrctPtr []*StrctBool
	}

	type StrctSet struct {
		M    map[string]bool
		Sl   []string
		Mptr *map[int]bool
	}

	type StrctEmbedDuration struct {
		time.Duration
	}

	type StrctEmbedDurationPtr struct {
		*time.Duration
	}

	type StrctStarval struct {
		Star            starlark.Value
		StarPtr         *starlark.Value
		Star2Ptr        **starlark.Value
		NotStar         dummyValue
		ExpandedStar    starlark.Callable
		ExpandedStarPtr *starlark.Callable
	}

	type StrctMy struct {
		Int       myInt
		Float     myFloat
		String    myString
		Bool      myBool
		IntPtr    *myInt
		FloatPtr  *myFloat
		StringPtr *myString
		BoolPtr   *myBool
	}

	cases := []struct {
		name string
		vals map[string]starlark.Value
		dst  any
		want any
		err  string
	}{
		{"None into *bool lower name", M{"bptr": starlark.None}, &StrctBool{}, StrctBool{Bptr: nil}, ""},
		{"None into *bool upper name", M{"Bptr": starlark.None}, &StrctBool{}, StrctBool{Bptr: nil}, ""},
		{"None into non-pointer", M{"I": starlark.None}, &StrctNums{}, nil, `I: cannot convert Starlark NoneType to Go type int`},
		{"None into slice", M{"i": starlark.None}, &StrctList{}, StrctList{I: nil}, ""},
		{"None into map", M{"m": starlark.None}, &StrctSet{}, StrctSet{M: nil}, ""},
		{"Nil starlark value into *bool", M{"bptr": nil}, &StrctBool{}, nil, `Bptr: cannot convert nil Starlark value to Go type *bool`},
		{"Unknown starlark value into *bool", M{"bptr": dummyValue{starlark.None}}, &StrctBool{}, nil, `Bptr: cannot convert Starlark dummy to Go type *bool`},

		{"true into *bool", M{"bptr": starlark.Bool(true)}, &StrctBool{}, StrctBool{Bptr: &truev}, ""},
		{"true into bool", M{"B": starlark.Bool(true)}, &StrctBool{}, StrctBool{B: true}, ""},
		{"false into bool", M{"b": starlark.Bool(false)}, &StrctBool{}, StrctBool{B: false}, ""},
		{"false into *bool", M{"bptr": starlark.Bool(false)}, &StrctBool{}, StrctBool{Bptr: &falsev}, ""},
		{"true into **bool", M{"b2ptr": starlark.Bool(true)}, &StrctBool{}, nil, `B2ptr: cannot convert Starlark bool to Go type **bool`},
		{"true into ignored and b", M{"b": starlark.Bool(true), "ignored": starlark.Bool(true)}, &StrctBool{}, StrctBool{B: true}, ``},
		{"true into *int", M{"iptr": starlark.Bool(true)}, &StrctNums{}, nil, `Iptr: cannot convert Starlark bool to Go type *int`},
		{"true into int", M{"i": starlark.Bool(true)}, &StrctNums{}, nil, `I: cannot convert Starlark bool to Go type int`},

		{"'a' into string", M{"s": starlark.String("a")}, &StrctStr{}, StrctStr{S: "a"}, ``},
		{"'a' into *string", M{"sptr": starlark.String("a")}, &StrctStr{}, StrctStr{Sptr: sptr("a")}, ``},
		{"'a' into **string", M{"s2ptr": starlark.String("a")}, &StrctStr{}, nil, `S2ptr: cannot convert Starlark string to Go type **string`},
		{"'a' into []byte", M{"bs": starlark.String("a")}, &StrctStr{}, StrctStr{Bs: []byte("a")}, ``},
		{"'a' into *[]byte", M{"bsptr": starlark.String("a")}, &StrctStr{}, StrctStr{BsPtr: bsptr("a")}, ``},
		{"'a' into **[]byte", M{"bs2ptr": starlark.String("a")}, &StrctStr{}, nil, `Bs2Ptr: cannot convert Starlark string to Go type **[]uint8`},
		{"'a' into *int", M{"iptr": starlark.String("a")}, &StrctNums{}, nil, `Iptr: cannot convert Starlark string to Go type *int`},
		{"'a' into int", M{"i": starlark.String("a")}, &StrctNums{}, nil, `I: cannot convert Starlark string to Go type int`},

		{"b'abc' into string", M{"s": starlark.Bytes("abc")}, &StrctStr{}, StrctStr{S: "abc"}, ``},
		{"b'abc' into *string", M{"sptr": starlark.Bytes("abc")}, &StrctStr{}, StrctStr{Sptr: sptr("abc")}, ``},
		{"b'abc' into **string", M{"s2ptr": starlark.Bytes("abc")}, &StrctStr{}, nil, `S2ptr: cannot convert Starlark bytes to Go type **string`},
		{"b'abc' into []byte", M{"bs": starlark.Bytes("abc")}, &StrctStr{}, StrctStr{Bs: []byte("abc")}, ``},
		{"b'abv' into *[]byte", M{"bsptr": starlark.Bytes("abc")}, &StrctStr{}, StrctStr{BsPtr: bsptr("abc")}, ``},
		{"b'abc' into **[]byte", M{"bs2ptr": starlark.Bytes("abc")}, &StrctStr{}, nil, `Bs2Ptr: cannot convert Starlark bytes to Go type **[]uint8`},
		{"b'abc' into *int", M{"iptr": starlark.Bytes("abc")}, &StrctNums{}, nil, `Iptr: cannot convert Starlark bytes to Go type *int`},
		{"b'abc' into int", M{"i": starlark.Bytes("abc")}, &StrctNums{}, nil, `I: cannot convert Starlark bytes to Go type int`},

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
		{"9 into *float64", M{"fptr": starlark.MakeUint(9)}, &StrctNums{}, StrctNums{Fptr: fptr(9)}, ``},
		{"None into *float64", M{"fptr": starlark.None}, &StrctNums{}, StrctNums{Fptr: nil}, ``},
		{"-1 into int", M{"i": starlark.MakeInt(-1)}, &StrctNums{}, StrctNums{I: -1}, ``},
		{"-2 into int8", M{"int8": starlark.MakeInt(-2)}, &StrctNums{}, StrctNums{I8: -2}, ``},
		{"-3 into int16", M{"int16": starlark.MakeInt(-3)}, &StrctNums{}, StrctNums{I16: -3}, ``},
		{"-4 into int32", M{"int32": starlark.MakeInt(-4)}, &StrctNums{}, StrctNums{I32: -4}, ``},
		{"-5 into int64", M{"int64": starlark.MakeInt(-5)}, &StrctNums{}, StrctNums{I64: -5}, ``},
		{"-1 into uint", M{"u": starlark.MakeInt(-1)}, &StrctNums{}, nil, `U: cannot assign Starlark int to Go type uint: value out of range`},
		{"-2 into uint8", M{"u8": starlark.MakeInt(-2)}, &StrctNums{}, nil, `U8: cannot assign Starlark int to Go type uint8: value out of range`},
		{"-3 into uint16", M{"u16": starlark.MakeInt(-3)}, &StrctNums{}, nil, `U16: cannot assign Starlark int to Go type uint16: value out of range`},
		{"-4 into uint32", M{"u32": starlark.MakeInt(-4)}, &StrctNums{}, nil, `U32: cannot assign Starlark int to Go type uint32: value out of range`},
		{"-5 into uint64", M{"u64": starlark.MakeInt(-5)}, &StrctNums{}, nil, `U64: cannot assign Starlark int to Go type uint64: value out of range`},
		{"-6 into uintptr", M{"up": starlark.MakeInt(-6)}, &StrctNums{}, nil, `Up: cannot assign Starlark int to Go type uintptr: value out of range`},
		{"1 into bool", M{"b": starlark.MakeInt(1)}, &StrctBool{}, nil, `B: cannot convert Starlark int to Go type bool`},
		{"1 into *bool", M{"bptr": starlark.MakeInt(1)}, &StrctBool{}, nil, `Bptr: cannot convert Starlark int to Go type *bool`},
		{"too big into int", M{"i": starlark.MakeUint(math.MaxUint64)}, &StrctNums{}, nil, `I: cannot assign Starlark int to Go type int: value out of range`},
		{"too big into int8", M{"int8": starlark.MakeInt(math.MaxInt8 + 1)}, &StrctNums{}, nil, `I8: cannot assign Starlark int to Go type int8: value out of range`},
		{"too big into int16", M{"int16": starlark.MakeInt(math.MaxInt16 + 1)}, &StrctNums{}, nil, `I16: cannot assign Starlark int to Go type int16: value out of range`},
		{"too big into int32", M{"int32": starlark.MakeInt(math.MaxInt32 + 1)}, &StrctNums{}, nil, `I32: cannot assign Starlark int to Go type int32: value out of range`},
		{"too big into int64", M{"int64": starlark.MakeUint(math.MaxInt64 + 1)}, &StrctNums{}, nil, `I64: cannot assign Starlark int to Go type int64: value out of range`},
		{"too big into uint", M{"u": starlark.MakeBigInt(tooBig)}, &StrctNums{}, nil, `U: cannot assign Starlark int to Go type uint: value out of range`},
		{"too big into uint8", M{"u8": starlark.MakeInt(math.MaxUint8 + 1)}, &StrctNums{}, nil, `U8: cannot assign Starlark int to Go type uint8: value out of range`},
		{"too big into uint16", M{"u16": starlark.MakeInt(math.MaxUint16 + 1)}, &StrctNums{}, nil, `U16: cannot assign Starlark int to Go type uint16: value out of range`},
		{"too big into uint32", M{"u32": starlark.MakeInt(math.MaxUint32 + 1)}, &StrctNums{}, nil, `U32: cannot assign Starlark int to Go type uint32: value out of range`},
		{"too big into uint64", M{"u64": starlark.MakeBigInt(tooBig)}, &StrctNums{}, nil, `U64: cannot assign Starlark int to Go type uint64: value out of range`},
		{"too big into uintptr", M{"up": starlark.MakeBigInt(tooBig)}, &StrctNums{}, nil, `Up: cannot assign Starlark int to Go type uintptr: value out of range`},

		{"1.1 into int", M{"i": starlark.Float(1.1)}, &StrctNums{}, nil, `I: cannot assign Starlark float to Go type int: value cannot be exactly represented`},
		{"1.1 into int8", M{"int8": starlark.Float(1.1)}, &StrctNums{}, nil, `I8: cannot assign Starlark float to Go type int8: value cannot be exactly represented`},
		{"1.1 into int16", M{"int16": starlark.Float(1.1)}, &StrctNums{}, nil, `I16: cannot assign Starlark float to Go type int16: value cannot be exactly represented`},
		{"1.1 into int32", M{"int32": starlark.Float(1.1)}, &StrctNums{}, nil, `I32: cannot assign Starlark float to Go type int32: value cannot be exactly represented`},
		{"1.1 into int64", M{"int64": starlark.Float(1.1)}, &StrctNums{}, nil, `I64: cannot assign Starlark float to Go type int64: value cannot be exactly represented`},
		{"1.1 into uint", M{"U": starlark.Float(1.1)}, &StrctNums{}, nil, `U: cannot assign Starlark float to Go type uint: value cannot be exactly represented`},
		{"1.1 into uint8", M{"U8": starlark.Float(1.1)}, &StrctNums{}, nil, `U8: cannot assign Starlark float to Go type uint8: value cannot be exactly represented`},
		{"1.1 into uint16", M{"U16": starlark.Float(1.1)}, &StrctNums{}, nil, `U16: cannot assign Starlark float to Go type uint16: value cannot be exactly represented`},
		{"1.1 into uint32", M{"U32": starlark.Float(1.1)}, &StrctNums{}, nil, `U32: cannot assign Starlark float to Go type uint32: value cannot be exactly represented`},
		{"1.1 into uint64", M{"U64": starlark.Float(1.1)}, &StrctNums{}, nil, `U64: cannot assign Starlark float to Go type uint64: value cannot be exactly represented`},
		{"1.1 into uintptr", M{"up": starlark.Float(1.1)}, &StrctNums{}, nil, `Up: cannot assign Starlark float to Go type uintptr: value cannot be exactly represented`},
		{"1.1 into float32", M{"f32": starlark.Float(1.1)}, &StrctNums{}, StrctNums{F32: 1.1}, ``},
		{"1.1 into float64", M{"f64": starlark.Float(1.1)}, &StrctNums{}, StrctNums{F64: 1.1}, ``},
		{"1.1 into *float64", M{"fptr": starlark.Float(1.1)}, &StrctNums{}, StrctNums{Fptr: fptr(1.1)}, ``},

		{"-1.1 into int", M{"i": starlark.Float(-1.1)}, &StrctNums{}, nil, `I: cannot assign Starlark float to Go type int: value cannot be exactly represented`},
		{"-1.1 into int8", M{"int8": starlark.Float(-1.1)}, &StrctNums{}, nil, `I8: cannot assign Starlark float to Go type int8: value cannot be exactly represented`},
		{"-1.1 into int16", M{"int16": starlark.Float(-1.1)}, &StrctNums{}, nil, `I16: cannot assign Starlark float to Go type int16: value cannot be exactly represented`},
		{"-1.1 into int32", M{"int32": starlark.Float(-1.1)}, &StrctNums{}, nil, `I32: cannot assign Starlark float to Go type int32: value cannot be exactly represented`},
		{"-1.1 into int64", M{"int64": starlark.Float(-1.1)}, &StrctNums{}, nil, `I64: cannot assign Starlark float to Go type int64: value cannot be exactly represented`},
		{"-1.1 into uint", M{"U": starlark.Float(-1.1)}, &StrctNums{}, nil, `U: cannot assign Starlark float to Go type uint: value cannot be exactly represented`},
		{"-1.1 into uint8", M{"U8": starlark.Float(-1.1)}, &StrctNums{}, nil, `U8: cannot assign Starlark float to Go type uint8: value cannot be exactly represented`},
		{"-1.1 into uint16", M{"U16": starlark.Float(-1.1)}, &StrctNums{}, nil, `U16: cannot assign Starlark float to Go type uint16: value cannot be exactly represented`},
		{"-1.1 into uint32", M{"U32": starlark.Float(-1.1)}, &StrctNums{}, nil, `U32: cannot assign Starlark float to Go type uint32: value cannot be exactly represented`},
		{"-1.1 into uint64", M{"U64": starlark.Float(-1.1)}, &StrctNums{}, nil, `U64: cannot assign Starlark float to Go type uint64: value cannot be exactly represented`},
		{"-1.1 into uintptr", M{"Up": starlark.Float(-1.1)}, &StrctNums{}, nil, `Up: cannot assign Starlark float to Go type uintptr: value cannot be exactly represented`},
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
		{"-1.0 into uint", M{"U": starlark.Float(-1.0)}, &StrctNums{}, nil, `U: cannot assign Starlark float to Go type uint: value out of range`},
		{"-1.0 into uint8", M{"U8": starlark.Float(-1.0)}, &StrctNums{}, nil, `U8: cannot assign Starlark float to Go type uint8: value out of range`},
		{"-1.0 into uint16", M{"U16": starlark.Float(-1.0)}, &StrctNums{}, nil, `U16: cannot assign Starlark float to Go type uint16: value out of range`},
		{"-1.0 into uint32", M{"U32": starlark.Float(-1.0)}, &StrctNums{}, nil, `U32: cannot assign Starlark float to Go type uint32: value out of range`},
		{"-1.0 into uint64", M{"U64": starlark.Float(-1.0)}, &StrctNums{}, nil, `U64: cannot assign Starlark float to Go type uint64: value out of range`},
		{"-1.0 into uintptr", M{"Up": starlark.Float(-1.0)}, &StrctNums{}, nil, `Up: cannot assign Starlark float to Go type uintptr: value out of range`},
		{"-1.0 into float32", M{"f32": starlark.Float(-1.0)}, &StrctNums{}, StrctNums{F32: -1.0}, ``},
		{"-1.0 into float64", M{"f64": starlark.Float(-1.0)}, &StrctNums{}, StrctNums{F64: -1.0}, ``},
		{"-1.0 into *bool", M{"bptr": starlark.Float(-1.0)}, &StrctBool{}, nil, `Bptr: cannot convert Starlark float to Go type *bool`},
		{"-1.0 into bool", M{"b": starlark.Float(-1.0)}, &StrctBool{}, nil, `B: cannot convert Starlark float to Go type bool`},

		{"too big into int", M{"i": starlark.Float(math.MaxInt + 1)}, &StrctNums{}, nil, `I: cannot assign Starlark float to Go type int: value out of range`},
		{"too big into int8", M{"int8": starlark.Float(math.MaxInt8 + 1)}, &StrctNums{}, nil, `I8: cannot assign Starlark float to Go type int8: value out of range`},
		{"too big into int16", M{"int16": starlark.Float(math.MaxInt16 + 1)}, &StrctNums{}, nil, `I16: cannot assign Starlark float to Go type int16: value out of range`},
		{"too big into int32", M{"int32": starlark.Float(math.MaxInt32 + 1)}, &StrctNums{}, nil, `I32: cannot assign Starlark float to Go type int32: value out of range`},
		{"too big into int64", M{"int64": starlark.Float(math.MaxInt64 + 1)}, &StrctNums{}, nil, `I64: cannot assign Starlark float to Go type int64: value out of range`},
		{"too big into uint", M{"U": starlark.Float(math.MaxUint + 1)}, &StrctNums{}, nil, `U: cannot assign Starlark float to Go type uint: value out of range`},
		{"too big into uint8", M{"U8": starlark.Float(math.MaxUint8 + 1)}, &StrctNums{}, nil, `U8: cannot assign Starlark float to Go type uint8: value out of range`},
		{"too big into uint16", M{"U16": starlark.Float(math.MaxUint16 + 1)}, &StrctNums{}, nil, `U16: cannot assign Starlark float to Go type uint16: value out of range`},
		{"too big into uint32", M{"U32": starlark.Float(math.MaxUint32 + 1)}, &StrctNums{}, nil, `U32: cannot assign Starlark float to Go type uint32: value out of range`},
		{"too big into uintptr", M{"Up": starlark.Float(math.MaxUint64 + 1)}, &StrctNums{}, nil, `Up: cannot assign Starlark float to Go type uintptr: value out of range`},
		{"too big into float32", M{"f32": starlark.Float(math.MaxFloat64)}, &StrctNums{}, nil, `F32: cannot assign Starlark float to Go type float32: value cannot be exactly represented`},
		{"too big into float64", M{"f64": starlark.Float(math.MaxFloat64 + 1)}, &StrctNums{}, StrctNums{F64: math.MaxFloat64 + 1}, ``},
		{"too big for exact float32", M{"f32": starlark.MakeUint(9007199254740993)}, &StrctNums{}, nil, `F32: cannot assign Starlark int to Go type float32: value cannot be exactly represented`},
		{"too big for exact float64", M{"f64": starlark.MakeUint(9007199254740993)}, &StrctNums{}, nil, `F64: cannot assign Starlark int to Go type float64: value cannot be exactly represented`},
		{"very big int to float32", M{"f32": starlark.MakeUint(9007199254740992)}, &StrctNums{}, StrctNums{F32: 9007199254740992}, ``},
		{"very big int to float64", M{"f64": starlark.MakeUint(9007199254740992)}, &StrctNums{}, StrctNums{F64: 9007199254740992}, ``},
		{"very small int to float32", M{"f32": starlark.MakeInt(-9007199254740992)}, &StrctNums{}, StrctNums{F32: -9007199254740992}, ``},
		{"very small int to float64", M{"f64": starlark.MakeInt(-9007199254740992)}, &StrctNums{}, StrctNums{F64: -9007199254740992}, ``},
		{"too small int to float32", M{"f32": starlark.MakeInt(-9007199254740993)}, &StrctNums{}, nil, `F32: cannot assign Starlark int to Go type float32: value cannot be exactly represented`},
		{"too small int to float64", M{"f64": starlark.MakeInt(-9007199254740993)}, &StrctNums{}, nil, `F64: cannot assign Starlark int to Go type float64: value cannot be exactly represented`},
		{"too big bigint to float32", M{"f32": starlark.MakeBigInt(tooBig)}, &StrctNums{}, nil, `F32: cannot assign Starlark int to Go type float32: value cannot be exactly represented`},
		{"too big bigint to float64", M{"f64": starlark.MakeBigInt(tooBig)}, &StrctNums{}, nil, `F64: cannot assign Starlark int to Go type float64: value cannot be exactly represented`},

		{"embedded ptr int", M{"i": starlark.MakeInt(1)}, &StrctDict{}, StrctDict{StrctNums: &StrctNums{I: 1}}, ``},
		{"embedded ptr *int", M{"iptr": starlark.MakeInt(1)}, &StrctDict{}, StrctDict{StrctNums: &StrctNums{Iptr: iptr(1)}}, ``},
		{"embedded ptr string", M{"s": starlark.String("abc")}, &StrctDict{}, StrctDict{StrctStr: StrctStr{S: "abc"}}, ``},
		{"embedded ptr *string", M{"sptr": starlark.String("abc")}, &StrctDict{}, StrctDict{StrctStr: StrctStr{Sptr: sptr("abc")}}, ``},
		{"embedded ptr **string", M{"s2ptr": starlark.String("abc")}, &StrctDict{}, nil, `StrctStr.S2ptr: cannot convert Starlark string to Go type **string`},
		{"embedded ptr unprefixed b", M{"B": starlark.Bool(true)}, &StrctDict{}, StrctDict{}, ``},
		{"embedded ptr prefixed b", M{"bools": dict(M{"B": starlark.Bool(true)})}, &StrctDict{}, StrctDict{StrctBool: StrctBool{B: true}}, ``},
		{"embedded ptr prefixed *bool", M{"bools": dict(M{"bptr": starlark.Bool(true)})}, &StrctDict{}, StrctDict{StrctBool: StrctBool{Bptr: &truev}}, ``},
		{"embedded ptr prefixed **bool", M{"bools": dict(M{"b2ptr": starlark.Bool(true)})}, &StrctDict{}, nil, `StrctBool.B2ptr: cannot convert Starlark bool to Go type **bool`},

		{"list int", M{"i": list(starlark.MakeInt(1), starlark.MakeInt(2), starlark.MakeInt(3))}, &StrctList{}, StrctList{I: []int{1, 2, 3}}, ``},
		{"list *[]*int", M{"ptriptr": list(starlark.MakeInt(1), starlark.MakeInt(2), starlark.MakeInt(3))}, &StrctList{}, StrctList{PtrIptr: &[]*int{iptr(1), iptr(2), iptr(3)}}, ``},
		{"list string", M{"s": list(starlark.String("a"), starlark.String("b"))}, &StrctList{}, StrctList{S: []string{"a", "b"}}, ``},
		{"list empty *string", M{"sptr": list()}, &StrctList{}, StrctList{Sptr: []*string{}}, ``},
		{"list empty *[]*int", M{"ptriptr": list()}, &StrctList{}, StrctList{PtrIptr: &[]*int{}}, ``},
		{"list StrctBool", M{"strct": list(dict(M{"B": starlark.Bool(true)}), dict(M{"Bptr": starlark.Bool(true)}))}, &StrctList{}, StrctList{Strct: []StrctBool{{B: true}, {Bptr: &truev}}}, ``},
		{"list *StrctBool", M{"strctptr": list(dict(M{"B": starlark.Bool(true)}), dict(M{"Bptr": starlark.Bool(false)}))}, &StrctList{}, StrctList{StrctPtr: []*StrctBool{{B: true}, {Bptr: &falsev}}}, ``},
		{"list mixed values", M{"s": list(starlark.String("a"), starlark.MakeInt(1))}, &StrctList{}, nil, `S[1]: cannot convert Starlark int to Go type string`},
		{"list None *StrctBool", M{"strctptr": list(starlark.None, dict(M{"Bptr": starlark.Bool(true)}))}, &StrctList{}, StrctList{StrctPtr: []*StrctBool{nil, {Bptr: &truev}}}, ``},
		{"list None *string", M{"sptr": list(starlark.None, starlark.None)}, &StrctList{}, StrctList{Sptr: []*string{nil, nil}}, ``},
		{"list into non-slice", M{"b": list(starlark.Bool(true), starlark.Bool(false))}, &StrctBool{}, nil, `B: cannot convert Starlark list to Go type bool`},
		{"list into non-slice pointer", M{"bptr": list(starlark.Bool(true), starlark.Bool(false))}, &StrctBool{}, nil, `Bptr: cannot convert Starlark list to Go type *bool`},

		{"tuple int", M{"i": tup(starlark.MakeInt(1), starlark.MakeInt(2), starlark.MakeInt(3))}, &StrctList{}, StrctList{I: []int{1, 2, 3}}, ``},
		{"tuple string", M{"s": tup(starlark.String("a"), starlark.String("b"), starlark.String("c"))}, &StrctList{}, StrctList{S: []string{"a", "b", "c"}}, ``},
		{"tuple mixed values", M{"s": tup(starlark.String("a"), starlark.MakeInt(1))}, &StrctList{}, nil, `S[1]: cannot convert Starlark int to Go type string`},

		{"set into map", M{"m": set(starlark.String("a"), starlark.String("b"))}, &StrctSet{}, StrctSet{M: map[string]bool{"a": true, "b": true}}, ``},
		{"set into existing map", M{"m": set(starlark.String("a"), starlark.String("b"))}, &StrctSet{M: map[string]bool{"b": true, "c": true}}, StrctSet{M: map[string]bool{"a": true, "b": true, "c": true}}, ``},
		{"set into slice", M{"sl": set(starlark.String("a"), starlark.String("b"))}, &StrctSet{}, StrctSet{Sl: []string{"a", "b"}}, ``},
		{"set into existing slice", M{"sl": set(starlark.String("a"), starlark.String("b"))}, &StrctSet{Sl: []string{"c"}}, StrctSet{Sl: []string{"a", "b"}}, ``},
		{"set into longer existing slice", M{"sl": set(starlark.String("a"), starlark.String("b"))}, &StrctSet{Sl: []string{"c", "d", "e"}}, StrctSet{Sl: []string{"a", "b"}}, ``},
		{"empty set into existing slice", M{"sl": set()}, &StrctSet{Sl: []string{"c"}}, StrctSet{Sl: []string{}}, ``},
		{"set into *map", M{"mptr": set(starlark.MakeInt(1), starlark.MakeInt(2))}, &StrctSet{}, StrctSet{Mptr: &map[int]bool{1: true, 2: true}}, ``},
		{"None into *map", M{"mptr": starlark.None}, &StrctSet{Mptr: &map[int]bool{}}, StrctSet{Mptr: nil}, ``},
		{"set mixed values", M{"m": set(starlark.String("a"), starlark.MakeInt(1))}, &StrctSet{}, nil, `M[1]: cannot convert Starlark int to Go type string`},
		{"set into non-map", M{"b": set(starlark.String("a"), starlark.String("b"))}, &StrctBool{}, nil, `B: cannot convert Starlark set to Go type bool`},
		{"set into non-map pointer", M{"bptr": set(starlark.String("a"), starlark.String("b"))}, &StrctBool{}, nil, `Bptr: cannot convert Starlark set to Go type *bool`},

		{"decode into starlark value", M{"star": starlark.None}, &StrctStarval{}, StrctStarval{Star: starlark.None}, ``},
		{"decode into starlark value pointer", M{"starptr": starlark.MakeInt(1)}, &StrctStarval{}, StrctStarval{StarPtr: starptr(starlark.MakeInt(1))}, ``},
		{"decode into starlark **Value", M{"star2ptr": starlark.MakeInt(1)}, &StrctStarval{}, nil, `Star2Ptr: cannot convert Starlark int to Go type **starlark.Value`},
		{"decode into wrapped starlark value interface", M{"notstar": starlark.MakeInt(1)}, &StrctStarval{}, nil, `NotStar: cannot convert Starlark int to Go type starstruct.dummyValue`},
		{"decode into embedded starlark value", M{"anything": starlark.MakeInt(1)}, &dummyValue{}, nil, `Value: cannot convert Starlark StringDict to Go type starlark.Value`},
		{"decode into expanded starlark value interface", M{"expandedstar": starlark.MakeInt(1)}, &StrctStarval{}, nil, `ExpandedStar: cannot convert Starlark int to Go type starlark.Callable`},
		{"decode into expanded pointer to starlark value interface", M{"expandedstarptr": starlark.MakeInt(1)}, &StrctStarval{}, nil, `ExpandedStarPtr: cannot convert Starlark int to Go type *starlark.Callable`},

		{"target is embedded non-struct", M{"duration": starlark.MakeInt(1)}, &StrctEmbedDuration{}, nil, `Duration: cannot convert Starlark StringDict to Go type time.Duration`},
		{"target is embedded non-struct pointer", M{"duration": starlark.MakeInt(1)}, &StrctEmbedDurationPtr{}, nil, `Duration: cannot convert Starlark StringDict to Go type *time.Duration`},

		{"multiple errors partial decode",
			M{"int64": starlark.MakeInt(1), "U8": starlark.MakeInt(-2), "s2ptr": starlark.String("a"), "bools": dict(M{"b": starlark.True})},
			&StrctDict{},
			StrctDict{StrctNums: &StrctNums{I64: 1}, StrctBool: StrctBool{B: true}},
			`StrctNums.U8: cannot assign Starlark int to Go type uint8: value out of range
StrctStr.S2ptr: cannot convert Starlark string to Go type **string`},

		{"true into myBool", M{"bool": starlark.Bool(true)}, &StrctMy{}, StrctMy{Bool: true}, ""},
		{"true into *myBool", M{"boolptr": starlark.Bool(true)}, &StrctMy{}, StrctMy{BoolPtr: myTruePtr}, ""},
		{"string into myString", M{"string": starlark.String("abc")}, &StrctMy{}, StrctMy{String: "abc"}, ""},
		{"string into *myString", M{"stringptr": starlark.String("def")}, &StrctMy{}, StrctMy{StringPtr: (*myString)(sptr("def"))}, ""},
		{"int into myInt", M{"int": starlark.MakeInt(-123)}, &StrctMy{}, StrctMy{Int: -123}, ""},
		{"int into *myInt", M{"intptr": starlark.MakeInt(456)}, &StrctMy{}, StrctMy{IntPtr: (*myInt)(iptr(456))}, ""},
		{"int into myFloat", M{"float": starlark.MakeInt(-123)}, &StrctMy{}, StrctMy{Float: -123}, ""},
		{"int into *myFloat", M{"floatptr": starlark.MakeInt(456)}, &StrctMy{}, StrctMy{FloatPtr: (*myFloat)(fptr(456))}, ""},
		{"float into myFloat", M{"float": starlark.Float(-123)}, &StrctMy{}, StrctMy{Float: -123}, ""},
		{"float into *myFloat", M{"floatptr": starlark.Float(456)}, &StrctMy{}, StrctMy{FloatPtr: (*myFloat)(fptr(456))}, ""},
		{"float into myInt", M{"int": starlark.Float(-123)}, &StrctMy{}, StrctMy{Int: -123}, ""},
		{"float into *myInt", M{"intptr": starlark.Float(456)}, &StrctMy{}, StrctMy{IntPtr: (*myInt)(iptr(456))}, ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := FromStarlark(c.vals, c.dst)
			if c.err != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), c.err)
			} else {
				require.NoError(t, err)
			}

			if c.want != nil {
				rv := reflect.ValueOf(c.dst)
				require.Equal(t, c.want, rv.Elem().Interface())
			}
		})
	}
}

func TestFromStarlark_InvalidDestination(t *testing.T) {
	var s string

	require.PanicsWithValue(t, `destination value is not a pointer to a struct: string`, func() {
		_ = FromStarlark(nil, s)
	})
	require.PanicsWithValue(t, `destination value is not a pointer to a struct: *string`, func() {
		_ = FromStarlark(nil, &s)
	})
	require.PanicsWithValue(t, `destination value is not a pointer to a struct: nil`, func() {
		_ = FromStarlark(nil, nil)
	})

	type T struct{ I int }
	require.PanicsWithValue(t, `destination value is a nil pointer: *starstruct.T`, func() {
		_ = FromStarlark(nil, (*T)(nil))
	})
}

func TestFromStarlark_MaxFromErrors(t *testing.T) {
	type S struct {
		I  int
		S  string
		B  **bool
		Ch chan byte
	}

	t.Run("too many", func(t *testing.T) {
		var s S
		err := FromStarlark(M{
			"I":  starlark.MakeInt(1),
			"B":  starlark.True,
			"Ch": starlark.String("a"),
			"S":  starlark.MakeInt(32),
		}, &s, MaxFromErrors(2))

		require.Error(t, err)
		errs := err.(interface{ Unwrap() []error }).Unwrap()
		require.Len(t, errs, 3)

		var te *TypeError
		require.ErrorAs(t, errs[0], &te)
		require.Contains(t, errs[0].Error(), `S: cannot convert Starlark int to Go type string`)
		require.ErrorAs(t, errs[1], &te)
		require.Contains(t, errs[1].Error(), `B: cannot convert Starlark bool to Go type **bool`)
		require.ErrorAs(t, errs[1], &te)
		require.Contains(t, errs[2].Error(), `maximum number of errors reached`)
	})

	t.Run("exactly", func(t *testing.T) {
		var s S
		err := FromStarlark(M{
			"I":  starlark.MakeInt(1),
			"B":  starlark.True,
			"Ch": starlark.String("a"),
			"S":  starlark.MakeInt(32),
		}, &s, MaxFromErrors(3))

		require.Error(t, err)
		errs := err.(interface{ Unwrap() []error }).Unwrap()
		require.Len(t, errs, 3)

		var te *TypeError
		require.ErrorAs(t, errs[0], &te)
		require.Contains(t, errs[0].Error(), `S: cannot convert Starlark int to Go type string`)
		require.ErrorAs(t, errs[1], &te)
		require.Contains(t, errs[1].Error(), `B: cannot convert Starlark bool to Go type **bool`)
		require.ErrorAs(t, errs[1], &te)
		require.Contains(t, errs[2].Error(), `Ch: cannot convert Starlark string to Go type chan uint8`)
	})
}

func TestFromStarlark_DuplicateTarget(t *testing.T) {
	type S struct {
		I   int `starlark:"int"`
		Int *int
	}
	var s S
	err := FromStarlark(M{"int": starlark.MakeInt(123)}, &s)
	require.NoError(t, err)
	require.Equal(t, S{I: 123, Int: iptr(123)}, s)
}

func TestFromStarlark_CustomConverter(t *testing.T) {
	timet := reflect.TypeOf(time.Now())
	durt := reflect.TypeOf(time.Second)

	// custom converter that supports:
	// - string to time.Duration
	// - string to time.Time (yyyy-MM-dd)
	// - int to time.Duration (number of seconds)
	// - int to time.Time (unix epoch)
	// - leaves anything else alone
	customFn := func(path string, starv starlark.Value, gov reflect.Value) (bool, error) {
		got := gov.Type()
		if got != timet && got != durt {
			return false, nil
		}

		switch v := starv.(type) {
		case starlark.String:
			if got == timet {
				t, err := time.Parse(time.DateOnly, string(v))
				if err != nil {
					return false, err
				}
				gov.Set(reflect.ValueOf(t))
				return true, nil
			}

			d, err := time.ParseDuration(string(v))
			if err != nil {
				return false, err
			}
			gov.Set(reflect.ValueOf(d))
			return true, nil

		case starlark.Int:
			i, ok := v.Int64()
			if got == timet {
				if !ok {
					return false, errors.New("integer out of range for unix epoch")
				}
				t := time.Unix(i, 0)
				gov.Set(reflect.ValueOf(t))
				return true, nil
			}

			if !ok {
				return false, errors.New("integer out of range for duration")
			}
			d := time.Second * time.Duration(i)
			gov.Set(reflect.ValueOf(d))
			return true, nil

		default:
			return false, fmt.Errorf("unsupported starlark type: %s", starv.Type())
		}
	}

	type D struct {
		D1 time.Duration
		D2 *time.Duration
		Dn time.Duration `starlark:"-"`
		Ds []time.Duration
		B  bool
	}
	type T struct {
		T1 time.Time
		T2 *time.Time
		T3 time.Time
		T4 time.Time
		S  string
	}
	type S struct {
		D3 time.Duration
		D
		*T
		N D
	}

	var s S
	s.D.D1 = time.Hour
	s.D.Dn = time.Minute

	err := FromStarlark(M{
		"D3": starlark.String("12s"),
		"D1": starlark.MakeInt(25),
		"D2": starlark.MakeInt(30),
		"Dn": starlark.String("1s"),
		"Ds": list(starlark.MakeInt(33), starlark.String("2s")),
		"B":  starlark.True,
		"T1": starlark.String("2022-01-02"),
		"T2": starlark.MakeInt(1675613116),
		"T3": starlark.MakeInt(1672578000),
		"T4": starlark.String("not-a-date"),
		"S":  starlark.String("a"),
		"N":  dict(M{"D1": starlark.String("not-a-dur")}),
	}, &s, CustomFromConverter(customFn))
	require.Error(t, err)

	require.Equal(t, S{
		D3: 12 * time.Second,
		D: D{
			D1: 25 * time.Second,
			D2: durptr(30),
			Dn: time.Minute,
			Ds: []time.Duration{33 * time.Second, 2 * time.Second},
			B:  true,
		},
		T: &T{
			T1: date(2022, 1, 2),
			T3: time.Unix(1672578000, 0),
			S:  "a",
		},
		N: D{},
	}, s)

	var convErr *CustomConvError
	var typeErr *TypeError
	errs := err.(interface{ Unwrap() []error }).Unwrap()
	require.Len(t, errs, 3)
	require.ErrorAs(t, errs[0], &typeErr)
	require.Equal(t, "T.T2", typeErr.Path)
	require.ErrorAs(t, errs[1], &convErr)
	require.Equal(t, "T.T4", convErr.Path)
	require.ErrorAs(t, errs[2], &convErr)
	require.Equal(t, "N.D1", convErr.Path)
}
