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
