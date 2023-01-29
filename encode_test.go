package starstruct

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

func TestToStarlark(t *testing.T) {
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
