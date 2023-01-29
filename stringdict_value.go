package starstruct

import (
	"errors"

	"go.starlark.net/starlark"
)

var _ = dictGetSetter((*stringDictValue)(nil))

type dictGetSetter interface {
	starlark.Value
	Get(k starlark.Value) (v starlark.Value, found bool, err error)
	SetKey(k, v starlark.Value) error
}

type stringDictValue struct {
	starlark.StringDict
}

func (v stringDictValue) Type() string          { return "stringDictValue" }
func (v stringDictValue) Truth() starlark.Bool  { return starlark.Bool(true) }
func (v stringDictValue) Hash() (uint32, error) { return 0, nil }
func (v stringDictValue) Get(k starlark.Value) (starlark.Value, bool, error) {
	s, ok := k.(starlark.String)
	if !ok {
		return nil, false, errors.New("stringDictValue key is not a string")
	}
	x, ok := v.StringDict[string(s)]
	return x, ok, nil
}
func (v stringDictValue) SetKey(k, x starlark.Value) error {
	s, ok := k.(starlark.String)
	if !ok {
		return errors.New("stringDictValue key is not a string")
	}
	v.StringDict[string(s)] = x
	return nil
}
