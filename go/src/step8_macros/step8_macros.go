package main

import (
	"fmt"
	"os"
	"strings"

	"printer"
	"reader"
	"readline"
	"types"
)

func Read(raw string) (*types.Data, error) {
	return reader.ReadStr(raw)
}

var specialForms = map[string]func(list []*types.Data, env *types.Env) (*types.Data, error){}

func Eval(ast *types.Data, env *types.Env) (*types.Data, error) {
	for {
		if ast.List != nil {
			list := *ast.List
			if len(list) == 0 {
				return ast, nil
			}

			// Some special forms are implemented in place, since they support TCO.
			if list[0].Symbol != nil {
				// Try to expand macros first thing.
				var err error
				ast, err = macroexpand(ast, env)
				if err != nil {
					return nil, err
				}

				if ast.List == nil {
					return eval_ast(ast, env)
				}
				list = *ast.List

				sym := *list[0].Symbol
				switch sym {

				case "macroexpand":
					return macroexpand(list[1], env)

				case "quasiquote":
					ast = quasiquote(list[1])
					continue

				case "let*":
					// Second parameter should be a list of odd/even pairs.
					if list[1].List == nil {
						return nil, fmt.Errorf("First parameter of let* must be a list")
					}

					bindings := *list[1].List
					if len(bindings)%2 != 0 {
						return nil, fmt.Errorf("let* bindings must come in pairs; found %d", len(bindings))
					}

					letEnv := types.NewEnv(env, nil, nil)
					for i := 0; i < len(bindings); i += 2 {
						if bindings[i].Symbol == nil {
							return nil, fmt.Errorf("left-hand binding must be a symbol")
						}

						sym := *bindings[i].Symbol
						value := bindings[i+1]
						evald, err := Eval(value, letEnv)
						if err != nil {
							return nil, err
						}

						letEnv.Set(sym, evald)
					}

					//return Eval(list[2], letEnv)
					ast = list[2]
					env = letEnv
					continue

				case "do":
					parts := list[1 : len(list)-1] // Strip off the "do" and last value.
					_, err := eval_ast(&types.Data{List: &parts}, env)
					if err != nil {
						return nil, err
					}
					ast = list[len(list)-1]
					continue

				case "if":
					cond, err := Eval(list[1], env)
					if err != nil {
						return nil, err
					}
					if cond == types.False || cond == types.Nil {
						if len(list) <= 3 {
							return types.Nil, nil
						}
						ast = list[3]
						continue
					}

					if len(list) <= 2 {
						return types.Nil, nil
					}
					ast = list[2]
					continue
				}

				// If we're still here, try the special forms map.
				if sf, ok := specialForms[sym]; ok {
					return sf(list, env)
				}
			}

			evald, err := eval_ast(ast, env)
			if err != nil {
				return nil, err
			}

			elist := *evald.List
			if elist[0] == nil || (elist[0].Native == nil && elist[0].Closure == nil) {
				return nil, fmt.Errorf("cannot call non-function %v", elist[0])
			}
			if elist[0].Closure != nil {
				body, newEnv, err := callHelper(elist[0].Closure, elist[1:])
				if err != nil {
					return nil, err
				}
				ast = body
				env = newEnv
				continue // TCO
			}
			return elist[0].Native(elist[1:])
		}

		return eval_ast(ast, env)
	}
}

func Print(form *types.Data) string {
	return printer.PrintStr(form, true)
}

func rep(input string) (string, error) {
	form, err := Read(input)
	if err != nil {
		return "", err
	}

	evald, err := Eval(form, repl_env)
	if err != nil {
		return "", err
	}

	return Print(evald), nil
}

func eval_ast(ast *types.Data, env *types.Env) (*types.Data, error) {
	if ast.Symbol != nil {
		return env.Get(*ast.Symbol)
	}

	if ast.List != nil {
		evald, err := eval_list(*ast.List, env)
		if err != nil {
			return nil, err
		}
		return &types.Data{List: &evald}, nil
	}

	return ast, nil
}

func is_macro_call(ast *types.Data, env *types.Env) bool {
	if ast.List == nil {
		return false
	}

	list := *ast.List
	if len(list) == 0 || list[0].Symbol == nil {
		return false
	}

	sym := *list[0].Symbol
	if m := env.Find(sym); m != nil {
		if m.Closure != nil && m.Closure.IsMacro {
			return true
		}
	}

	return false
}

func macroexpand(ast *types.Data, env *types.Env) (*types.Data, error) {
	for is_macro_call(ast, env) {
		slc := *ast.List
		a0 := slc[0]
		mac, err := env.Get(*a0.Symbol)
		if err != nil {
			return nil, err
		}

		ast, err = apply(mac.Closure, slc[1:])
		if err != nil {
			return nil, err
		}
	}
	return ast, nil
}

func eval_list(list []*types.Data, env *types.Env) ([]*types.Data, error) {
	ret := []*types.Data{}
	for _, expr := range list {
		evald, err := Eval(expr, env)
		if err != nil {
			return nil, err
		}

		ret = append(ret, evald)
	}
	return ret, nil
}

var repl_env *types.Env

func main() {
	repl_env = types.NewEnv(nil, nil, nil)

	for key, val := range ns {
		repl_env.Set(key, &types.Data{Native: val})
	}

	specialForms["def!"] = sfDef
	specialForms["defmacro!"] = sfDefmacro
	specialForms["fn*"] = sfFn
	specialForms["quote"] = sfQuote

	// Execute functions defined in mal itself.
	for _, val := range nsMal {
		_, err := rep(val)
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
	}

	for {
		line, err := readline.Readline("user> ")
		if err != nil {
			break
		}
		line = strings.TrimRight(line, "\n")
		s, err := rep(line)
		if err != nil {
			fmt.Printf("%v\n", err)
		} else {
			fmt.Println(s)
		}
	}
}

// Implementations of the special forms.
func doDef(list []*types.Data, env *types.Env, fun string) (*types.Data, error) {
	if list[1].Symbol == nil {
		return nil, fmt.Errorf("First parameter for %s must be a symbol", fun)
	}

	name := *list[1].Symbol
	evald, err := Eval(list[2], env)
	if err != nil {
		return nil, err
	}

	env.Set(name, evald)
	return evald, nil
}

func sfDef(list []*types.Data, env *types.Env) (*types.Data, error) {
	return doDef(list, env, "def!")
}

func sfDefmacro(list []*types.Data, env *types.Env) (*types.Data, error) {
	f, err := doDef(list, env, "defmacro!")
	if err != nil {
		return nil, err
	}

	f.Closure.IsMacro = true
	return f, nil
}

func sfFn(list []*types.Data, env *types.Env) (*types.Data, error) {
	// Builds a new function closure.
	if list[1].List == nil {
		return nil, fmt.Errorf("Function parameters must be a list.")
	}

	c := &types.Closure{env, nil, "", list[2], false}
	for i, p := range *list[1].List {
		if p.Symbol == nil {
			return nil, fmt.Errorf("Function parameter must be a symbol.")
		}

		if *p.Symbol == "&" {
			if i != len(*list[1].List)-2 {
				return nil, fmt.Errorf("Exactly 1 value must follow a & in arg list; found %s", len(*list[1].List)-i-1)
			}

			tp := (*list[1].List)[i+1]
			if tp.Symbol == nil {
				return nil, fmt.Errorf("Tail parameter must be a symbol.")
			}

			c.TailParams = *tp.Symbol
			break
		}

		c.Params = append(c.Params, *p.Symbol)
	}
	return &types.Data{Closure: c}, nil
}

func sfQuote(list []*types.Data, env *types.Env) (*types.Data, error) {
	return list[1], nil
}

func quasiquote(ast *types.Data) *types.Data {
	if ast.List == nil || len(*ast.List) == 0 {
		quote := "quote"
		list := []*types.Data{&types.Data{Symbol: &quote}, ast}
		return &types.Data{List: &list}
	}

	slc := *ast.List
	a0 := slc[0]

	if a0.Symbol != nil && *a0.Symbol == "unquote" {
		return slc[1]
	}
	if a0.List != nil {
		slc0 := *a0.List
		a00 := slc0[0]
		if a00.Symbol != nil && *a00.Symbol == "splice-unquote" {
			return list(sym("concat"), slc0[1], quasiquote(list(slc[1:]...)))
		}
	}
	return list(sym("cons"), quasiquote(a0), quasiquote(list(slc[1:]...)))
}

func list(elements ...*types.Data) *types.Data {
	return &types.Data{List: &elements}
}

func sym(symbol string) *types.Data {
	return &types.Data{Symbol: &symbol}
}

/*
func quasiquote(ast *types.Data) *types.Data {
	if ast.List == nil {
		quote := "quote"
		list := []*types.Data{
			&types.Data{Symbol: &quote},
			ast,
		}
		return &types.Data{List: &list}
	}

	if len(*ast.List) == 0 {
		return ast
	}

	fst := (*ast.List)[0]
	if fst.Symbol != nil && *fst.Symbol == "unquote" {
		return (*ast.List)[1]
	}

	if fst.List != nil && (*fst.List)[0].Symbol != nil && *(*fst.List)[0].Symbol == "splice-unquote" {
		concat := "concat"
		tail := (*ast.List)[1:]
		list := []*types.Data{
			&types.Data{Symbol: &concat},
			(*fst.List)[1],
			quasiquote(&types.Data{List: &tail}),
		}
		return &types.Data{List: &list}
	}

	cons := "cons"
	tail := (*ast.List)[1:]
	list := []*types.Data{
		&types.Data{Symbol: &cons},
		quasiquote(fst),
		quasiquote(&types.Data{List: &tail}),
	}

	val := &types.Data{List: &list}
	return val
}
*/

// Given a list for a closure (or macro) call, builds the new environment, with
// everything bound properly for evaluating the function's body. Returns the
// body, environment, and error.
func callHelper(f *types.Closure, args []*types.Data) (*types.Data, *types.Env, error) {
	expected := len(f.Params)
	found := len(args)
	if found < expected {
		return nil, nil, fmt.Errorf("Not enough parameters: expected at least %d, got %d",
			expected, found)
	}

	newEnv := types.NewEnv(f.Env, f.Params, args[0:expected])

	if f.TailParams != "" {
		tailArgs := args[expected:]
		newEnv.Set(f.TailParams, &types.Data{List: &tailArgs})
	}

	return f.Body, newEnv, nil
}

func apply(f *types.Closure, args []*types.Data) (*types.Data, error) {
	body, env, err := callHelper(f, args)
	if err != nil {
		return nil, err
	}
	return Eval(body, env)
}
