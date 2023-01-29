package starstruct

import (
	"math"
	"math/big"

	"go.starlark.net/starlark"
)

type M = map[string]starlark.Value

func dict(m M) *starlark.Dict {
	d := starlark.NewDict(len(m))
	for k, v := range m {
		if err := d.SetKey(starlark.String(k), v); err != nil {
			panic(err)
		}
	}
	return d
}

func list(vs ...starlark.Value) *starlark.List {
	return starlark.NewList(vs)
}

func tup(vs ...starlark.Value) starlark.Tuple {
	return starlark.Tuple(vs)
}

func set(vs ...starlark.Value) *starlark.Set {
	x := starlark.NewSet(len(vs))
	for _, v := range vs {
		if err := x.Insert(v); err != nil {
			panic(err)
		}
	}
	return x
}

func sptr(s string) *string  { return &s }
func bsptr(s string) *[]byte { bs := []byte(s); return &bs }
func iptr(i int) *int        { return &i }

var (
	truev, falsev = true, false
	tooBig        = big.NewInt(1).Add(big.NewInt(1).SetUint64(math.MaxUint64), big.NewInt(1))
)
