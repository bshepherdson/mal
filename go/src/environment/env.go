package environment

import "fmt"
import "types"

type Env struct {
	data map[string]*types.Data
	outer *Env
}

func NewEnv(outer *Env) *Env {
	return &Env{map[string]*types.Data{}, outer}
}

func (e *Env) Set(key string, value *types.Data) {
	e.data[key] = value
}

func (e *Env) Find(key string) *types.Data {
	if value, ok := e.data[key]; ok {
		return value
	}
	if e.outer != nil {
		return e.outer.Find(key)
	}

	return nil
}

func (e *Env) Get(key string) (*types.Data, error) {
	value := e.Find(key)
	if value == nil {
		return nil, fmt.Errorf("not found: '%s'", key)
	}
	return value, nil
}
