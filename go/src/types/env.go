package types

import "fmt"

type Env struct {
	data  map[string]*Data
	outer *Env
}

func NewEnv(outer *Env, binds []string, exprs []*Data) *Env {
	if len(binds) != len(exprs) {
		panic("mismatched expressions")
	}

	env := &Env{map[string]*Data{}, outer}
	for i, expr := range exprs {
		env.Set(binds[i], expr)
	}

	return env
}

func (e *Env) Set(key string, value *Data) {
	e.data[key] = value
}

func (e *Env) Find(key string) *Data {
	if value, ok := e.data[key]; ok {
		return value
	}
	if e.outer != nil {
		return e.outer.Find(key)
	}

	return nil
}

func (e *Env) Get(key string) *Data {
	value := e.Find(key)
	if value == nil {
		s := fmt.Sprintf("not found: '%s'", key)
		return Throw(&Data{String: &s})
	}
	return value
}
