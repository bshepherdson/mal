package main

import (
	"fmt"
	"os"
	"strings"

	"printer"
	"reader"
	"readline"
	. "types"
)

func Read(raw string) *Data {
	return reader.ReadStr(raw)
}

var specialForms = map[string]func(list []*Data, env *Env) *Data{}

func Eval(ast *Data, env *Env) *Data {
	for {
		if HasError() {
			return nil
		}

		if ast.List != nil {
			list := *ast.List
			if len(list) == 0 {
				return ast
			}

			// Some special forms are implemented in place, since they support TCO.
			if list[0].Symbol != nil {
				// Try to expand macros first thing.
				ast = macroexpand(ast, env)
				if HasError() {
					return nil
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
						return Throw(str("First parameter of let* must be a list"))
					}

					bindings := *list[1].List
					if len(bindings)%2 != 0 {
						return Throw(str("let* bindings must come in pairs; found %d", len(bindings)))
					}

					letEnv := NewEnv(env, nil, nil)
					for i := 0; i < len(bindings); i += 2 {
						if bindings[i].Symbol == nil {
							return Throw(str("left-hand binding must be a symbol"))
						}

						sym := *bindings[i].Symbol
						value := bindings[i+1]
						evald := Eval(value, letEnv)
						if HasError() {
							return nil
						}

						letEnv.Set(sym, evald)
					}

					//return Eval(list[2], letEnv)
					ast = list[2]
					env = letEnv
					continue

				case "do":
					parts := list[1 : len(list)-1] // Strip off the "do" and last value.
					eval_ast(&Data{List: &parts}, env)
					if HasError() {
						return nil
					}
					ast = list[len(list)-1]
					continue

				case "if":
					cond := Eval(list[1], env)
					if HasError() {
						return nil
					}
					if cond == False || cond == Nil {
						if len(list) <= 3 {
							return Nil
						}
						ast = list[3]
						continue
					}

					if len(list) <= 2 {
						return Nil
					}
					ast = list[2]
					continue
				}

				// If we're still here, try the special forms map.
				if sf, ok := specialForms[sym]; ok {
					return sf(list, env)
				}
			}

			evald := eval_ast(ast, env)
			if HasError() {
				return nil
			}

			elist := *evald.List
			if elist[0] == nil || (elist[0].Native == nil && elist[0].Closure == nil) {
				return Throw(str("cannot call non-function %v", elist[0]))
			}
			if elist[0].Closure != nil {
				body, newEnv := callHelper(elist[0].Closure, elist[1:])
				if HasError() {
					return nil
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

func Print(form *Data) string {
	return printer.PrintStr(form, true)
}

func rep(input string) string {
	form := Read(input)
	var evald *Data

	if !HasError() {
		evald = Eval(form, repl_env)
	}

	if HasError() {
		err := GetError()
		if err.String != nil {
			return *err.String
		}
		return fmt.Sprintf("uncaught error: %s", Print(err))
	}
	return Print(evald)
}

func eval_ast(ast *Data, env *Env) *Data {
	if ast.Symbol != nil {
		return env.Get(*ast.Symbol)
	}

	if ast.List != nil {
		evald := eval_list(*ast.List, env)
		if HasError() {
			return nil
		}
		return &Data{List: &evald}
	}

	return ast
}

func is_macro_call(ast *Data, env *Env) bool {
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

func macroexpand(ast *Data, env *Env) *Data {
	for is_macro_call(ast, env) {
		slc := *ast.List
		a0 := slc[0]
		mac := env.Get(*a0.Symbol)
		if HasError() {
			return nil
		}

		ast = apply(mac.Closure, slc[1:])
		if HasError() {
			return nil
		}
	}
	return ast
}

func eval_list(list []*Data, env *Env) []*Data {
	ret := []*Data{}
	for _, expr := range list {
		evald := Eval(expr, env)
		if HasError() {
			return nil
		}

		ret = append(ret, evald)
	}
	return ret
}

var repl_env *Env

func main() {
	repl_env = NewEnv(nil, nil, nil)

	for key, val := range ns {
		repl_env.Set(key, &Data{Native: val})
	}

	specialForms["def!"] = sfDef
	specialForms["defmacro!"] = sfDefmacro
	specialForms["fn*"] = sfFn
	specialForms["quote"] = sfQuote

	// Execute functions defined in mal itself.
	for _, val := range nsMal {
		rep(val)
		if HasError() {
			fmt.Printf("%v\n", GetError())
			os.Exit(1)
		}
	}

	for {
		line, err := readline.Readline("user> ")
		if err != nil {
			break
		}
		line = strings.TrimRight(line, "\n")
		s := rep(line)
		if HasError() {
			fmt.Printf("uncaught error: %v\n", printer.PrintStr(GetError(), false))
		} else {
			fmt.Println(s)
		}
		ClearError()
	}
}

// Implementations of the special forms.
func doDef(list []*Data, env *Env, fun string) *Data {
	if list[1].Symbol == nil {
		return Throw(str("First parameter for %s must be a symbol", fun))
	}

	name := *list[1].Symbol
	evald := Eval(list[2], env)
	if HasError() {
		return nil
	}

	env.Set(name, evald)
	return evald
}

func sfDef(list []*Data, env *Env) *Data {
	return doDef(list, env, "def!")
}

func sfDefmacro(list []*Data, env *Env) *Data {
	f := doDef(list, env, "defmacro!")
	if HasError() {
		return nil
	}

	f.Closure.IsMacro = true
	return f
}

func sfFn(list []*Data, env *Env) *Data {
	// Builds a new function closure.
	if list[1].List == nil {
		return Throw(str("Function parameters must be a list."))
	}

	c := &Closure{env, nil, "", list[2], false}
	for i, p := range *list[1].List {
		if p.Symbol == nil {
			return Throw(str("Function parameter must be a symbol."))
		}

		if *p.Symbol == "&" {
			if i != len(*list[1].List)-2 {
				return Throw(str("Exactly 1 value must follow a & in arg list; found %s", len(*list[1].List)-i-1))
			}

			tp := (*list[1].List)[i+1]
			if tp.Symbol == nil {
				return Throw(str("Tail parameter must be a symbol."))
			}

			c.TailParams = *tp.Symbol
			break
		}

		c.Params = append(c.Params, *p.Symbol)
	}
	return &Data{Closure: c}
}

func sfQuote(list []*Data, env *Env) *Data {
	return list[1]
}

func quasiquote(ast *Data) *Data {
	if ast.List == nil || len(*ast.List) == 0 {
		quote := "quote"
		list := []*Data{&Data{Symbol: &quote}, ast}
		return &Data{List: &list}
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

func list(elements ...*Data) *Data {
	return &Data{List: &elements}
}

func sym(symbol string) *Data {
	return &Data{Symbol: &symbol}
}

func str(s string, args ...interface{}) *Data {
	s2 := fmt.Sprintf(s, args...)
	return &Data{String: &s2}
}

// Given a list for a closure (or macro) call, builds the new environment, with
// everything bound properly for evaluating the function's body. Returns the
// body, environment, and error.
func callHelper(f *Closure, args []*Data) (*Data, *Env) {
	expected := len(f.Params)
	found := len(args)
	if found < expected {
		return Throw(str("Not enough parameters: expected at least %d, got %d",
			expected, found)), nil
	}

	newEnv := NewEnv(f.Env, f.Params, args[0:expected])

	if f.TailParams != "" {
		tailArgs := args[expected:]
		newEnv.Set(f.TailParams, &Data{List: &tailArgs})
	}

	return f.Body, newEnv
}

func apply(f *Closure, args []*Data) *Data {
	body, env := callHelper(f, args)
	if HasError() {
		return nil
	}
	return Eval(body, env)
}
