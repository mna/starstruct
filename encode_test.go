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
