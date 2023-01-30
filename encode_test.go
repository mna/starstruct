package starstruct

import (
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

func TestToStarlark(t *testing.T) {
	type EmptyStruct struct{}
	type IntStruct struct {
		I int
	}
	type NestedStruct struct {
		IntStruct
	}
	type ChanStruct struct {
		Ch chan int
	}

	cases := []struct {
		name string
		vals any
		dst  map[string]starlark.Value
		want map[string]starlark.Value
		err  string
	}{
		{"nil in nil dst", struct{ Bptr *bool }{}, nil, nil, ""},
		{"nil in empty dst", struct{ Bptr *bool }{}, M{}, M{"Bptr": starlark.None}, ""},
		{"nil in non-empty dst", struct{ Bptr *bool }{}, M{"A": starlark.String("a")}, M{"A": starlark.String("a"), "Bptr": starlark.None}, ""},
		{"nil overrides dst", struct{ Bptr *bool }{}, M{"Bptr": starlark.String("a")}, M{"Bptr": starlark.None}, ""},
		{"nil as **bool", struct{ B **bool }{}, M{}, nil, `unsupported Go type **bool at B`},

		{"true as bool", struct{ B bool }{B: true}, M{}, M{"B": starlark.Bool(true)}, ""},
		{"true/false as bool/*bool", struct {
			B  bool
			B2 *bool
		}{B: true, B2: &falsev}, M{}, M{"B": starlark.Bool(true), "B2": starlark.Bool(false)}, ""},
		{"Bool as **bool", struct{ B **bool }{}, M{}, nil, `unsupported Go type **bool at B`},

		{"Int as int", struct{ I int }{I: 1}, M{}, M{"I": starlark.MakeInt(1)}, ``},
		{"Int as int8", struct{ I int8 }{I: 1}, M{}, M{"I": starlark.MakeInt(1)}, ``},
		{"Int as int16", struct{ I int16 }{I: 1}, M{}, M{"I": starlark.MakeInt(1)}, ``},
		{"Int as int32", struct{ I int32 }{I: 1}, M{}, M{"I": starlark.MakeInt(1)}, ``},
		{"Int as int64", struct{ I int32 }{I: 1}, M{}, M{"I": starlark.MakeInt(1)}, ``},
		{"Uint as uint", struct{ U uint }{U: 1}, M{}, M{"U": starlark.MakeUint(1)}, ``},
		{"Uint as uint8", struct{ U uint8 }{U: 1}, M{}, M{"U": starlark.MakeUint(1)}, ``},
		{"Uint as uint16", struct{ U uint16 }{U: 1}, M{}, M{"U": starlark.MakeUint(1)}, ``},
		{"Uint as uint32", struct{ U uint32 }{U: 1}, M{}, M{"U": starlark.MakeUint(1)}, ``},
		{"Uint as uint64", struct{ U uint32 }{U: 1}, M{}, M{"U": starlark.MakeUint(1)}, ``},
		{"Uint as uintptr", struct{ U uintptr }{U: 1}, M{}, M{"U": starlark.MakeUint(1)}, ``},
		{"Float as float32", struct{ F float32 }{F: 3}, M{}, M{"F": starlark.Float(3)}, ``},
		{"Float as float64", struct{ F float64 }{F: 3}, M{}, M{"F": starlark.Float(3)}, ``},
		{"Int as *int", struct{ I *int }{I: iptr(1)}, M{}, M{"I": starlark.MakeInt(1)}, ``},
		{"Uint as *uint", struct{ U *uint }{U: uptr(1)}, M{}, M{"U": starlark.MakeInt(1)}, ``},

		{"[]byte as Bytes", struct{ B []byte }{B: []byte(`hello`)}, M{}, M{"B": starlark.Bytes("hello")}, ``},
		{"string as String", struct{ S string }{S: "hello"}, M{}, M{"S": starlark.String("hello")}, ``},
		{"*[]byte as Bytes", struct{ Bptr *[]byte }{Bptr: bsptr("world")}, M{}, M{"Bptr": starlark.Bytes("world")}, ``},
		{"*string as String", struct{ Sptr *string }{Sptr: sptr("world")}, M{}, M{"Sptr": starlark.String("world")}, ``},
		{"[]byte as String", struct {
			B []byte `starlark:"bass,asstring"`
		}{B: []byte(`hello`)}, M{}, M{"bass": starlark.String("hello")}, ``},
		{"*[]byte as String", struct {
			Bptr *[]byte `starlark:"bass,asstring"`
		}{Bptr: bsptr("world")}, M{}, M{"bass": starlark.String("world")}, ``},
		{"string as Bytes", struct {
			S string `starlark:"sasbytes,asbytes"`
		}{S: "world"}, M{}, M{"sasbytes": starlark.Bytes("world")}, ``},
		{"*string as Bytes", struct {
			Sptr *string `starlark:"sasbytes,asbytes"`
		}{Sptr: sptr("world")}, M{}, M{"sasbytes": starlark.Bytes("world")}, ``},
		{"[]byte as List", struct {
			B []byte `starlark:"baslist,aslist"`
		}{B: []byte{1, 2, 3}}, M{}, M{"baslist": list(starlark.MakeInt(1), starlark.MakeInt(2), starlark.MakeInt(3))}, ``},
		{"*[]byte as List", struct {
			Bptr *[]byte `starlark:"baslist,aslist"`
		}{Bptr: bsptr("\x01\x02\x03")}, M{}, M{"baslist": list(starlark.MakeInt(1), starlark.MakeInt(2), starlark.MakeInt(3))}, ``},
		{"[]*byte defaults as List", struct {
			Bptr []*byte `starlark:"baslist"`
		}{Bptr: []*byte{bptr(1), bptr(2)}}, M{}, M{"baslist": list(starlark.MakeInt(1), starlark.MakeInt(2))}, ``},

		{"[]int as List", struct{ Is []int }{Is: []int{1, 2, 3}}, M{}, M{"Is": list(starlark.MakeInt(1), starlark.MakeInt(2), starlark.MakeInt(3))}, ``},
		{"[]float as Tuple", struct {
			Fs []float64 `starlark:"fastup,astuple"`
		}{Fs: []float64{1, 2}}, M{}, M{"fastup": tup(starlark.Float(1), starlark.Float(2))}, ``},
		{"[]string as Set", struct {
			Ss []string `starlark:"sasset,asset"`
		}{Ss: []string{"a", "b"}}, M{}, M{"sasset": set(starlark.String("a"), starlark.String("b"))}, ``},
		{"[]string as Set of Bytes", struct {
			Ss []string `starlark:"sasset,asset,asbytes"`
		}{Ss: []string{"a", "b"}}, M{}, M{"sasset": set(starlark.Bytes("a"), starlark.Bytes("b"))}, ``},

		{"empty struct", &struct{}{}, M{}, M{}, ``},
		{"embedded struct no field", &struct{ EmptyStruct }{}, M{}, M{}, ``},
		{"embedded struct anonymous", &struct{ IntStruct }{IntStruct: IntStruct{I: 1}}, M{}, M{"I": starlark.MakeInt(1)}, ``},
		{"embedded struct named", &struct {
			IntStruct `starlark:"embedded"`
		}{IntStruct: IntStruct{I: 1}}, M{}, M{"embedded": dict(M{"I": starlark.MakeInt(1)})}, ``},
		{"nested embedded struct", &struct{ NestedStruct }{NestedStruct: NestedStruct{IntStruct: IntStruct{I: 2}}}, M{}, M{"I": starlark.MakeInt(2)}, ``},
		{"nested embedded struct named", &struct {
			NestedStruct `starlark:"nested"`
		}{NestedStruct: NestedStruct{IntStruct: IntStruct{I: 3}}}, M{}, M{"nested": dict(M{"I": starlark.MakeInt(3)})}, ``},
		{"nested embedded struct ignored", &struct {
			NestedStruct `starlark:"-"`
		}{NestedStruct: NestedStruct{IntStruct: IntStruct{I: 3}}}, M{}, M{}, ``},

		{"nil map", struct{ M map[string]bool }{}, M{}, M{"M": starlark.None}, ``},
		{"empty map", struct{ M map[string]bool }{M: map[string]bool{}}, M{}, M{"M": set()}, ``},
		{"map to set", struct{ M map[string]bool }{M: map[string]bool{"x": true}}, M{}, M{"M": set(starlark.String("x"))}, ``},
		{"map to set with false key", struct{ M map[string]bool }{M: map[string]bool{"x": true, "y": false}}, M{}, M{"M": set(starlark.String("x"))}, ``},
		{"nil *map", struct{ Mptr *map[string]bool }{}, M{}, M{"Mptr": starlark.None}, ``},

		{"time.Duration encodes as in64", struct{ Ts time.Duration }{Ts: time.Second}, M{}, M{"Ts": starlark.MakeInt(int(time.Second))}, ``},
		{"chan unsupported", struct{ Ch chan int }{Ch: make(chan int)}, M{}, nil, `unsupported Go type chan int at Ch`},
		{"chan unsupported ignored", struct {
			Ch chan int `starlark:"-"`
		}{Ch: make(chan int)}, M{}, M{}, ``},
		{"slice of funcs as tuples as sets", struct {
			Fs [][]func() `starlark:"fs,astuple,asset"`
		}{Fs: [][]func(){{func() {}}}}, M{}, nil, `unsupported Go type func() at Fs[0][0]`},
		{"slice of strings as tuple as set", struct {
			Ss [][]string `starlark:"ss,astuple,asset"`
		}{Ss: [][]string{{"a"}}}, M{}, M{"ss": tup(set(starlark.String("a")))}, ``},
		{"slice of strings as list as set as tuple", struct {
			Sss [][][]string `starlark:"sss,aslist,asset,astuple"`
		}{Sss: [][][]string{{{"a"}}}}, M{}, M{"sss": list(set(tup(starlark.String("a"))))}, ``},
		{"invalid slice type for as set", struct {
			Strct []struct{} `starlark:"strct,asset"`
		}{Strct: []struct{}{{}}}, M{}, nil, `failed to insert value into Set at Strct: unhashable type: dict`},
		{"invalid map key type for set", struct {
			M map[struct{}]bool
		}{M: map[struct{}]bool{{}: true}}, M{}, nil, `failed to insert value into Set at M: unhashable type: dict`},
		{"unsupported map key type", struct {
			M map[io.Reader]bool
		}{M: map[io.Reader]bool{io.Reader(nil): true}}, M{}, nil, `unsupported Go type io.Reader at M[<nil>]`},
		{"unsupported slice type", struct {
			Sl []chan int
		}{Sl: []chan int{make(chan int)}}, M{}, nil, `unsupported Go type chan int at Sl[0]`},
		{"unsupported struct field type", struct {
			Strct struct {
				Ch chan int
			}
		}{Strct: struct{ Ch chan int }{Ch: make(chan int)}}, M{}, nil, `unsupported Go type chan int at Strct.Ch`},
		{"unsupported embedded struct field type", struct {
			ChanStruct
		}{ChanStruct: ChanStruct{Ch: make(chan int)}}, M{}, nil, `unsupported Go type chan int at ChanStruct.Ch`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ToStarlark(c.vals, c.dst)
			if c.err != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), c.err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, c.want, c.dst)
		})
	}
}
