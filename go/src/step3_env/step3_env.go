package main

import (
	"fmt"
	"strings"

	"environment"
	"printer"
	"reader"
	"readline"
	"types"
)

var nilText string = "nil"
var trueText string = "true"
var falseText string = "false"

var lispNil = &types.Data{Symbol: &nilText}
var lispTrue = &types.Data{Symbol: &trueText}
var lispFalse = &types.Data{Symbol: &falseText}

func Read(raw string) (*types.Data, error) {
	return reader.ReadStr(raw)
}

func Eval(ast *types.Data, env *environment.Env) (*types.Data, error) {
	if ast.List != nil {
		list := *ast.List
		if len(list) == 0 {
			return ast, nil
		}

		if list[0].Symbol != nil {
			sym := *list[0].Symbol
			if sym == "def!" {
				if list[1].Symbol == nil {
					return nil, fmt.Errorf("First parameter for def! must be a symbol")
				}

				name := *list[1].Symbol
				evald, err := Eval(list[2], env)
				if err != nil {
					return nil, err
				}

				env.Set(name, evald)
				return evald, nil
			} else if sym == "let*" {
				// Second parameter should be a list of odd/even pairs.
				if list[1].List == nil {
					return nil, fmt.Errorf("First parameter of let* must be a list")
				}

				bindings := *list[1].List
				if len(bindings) % 2 != 0 {
					return nil, fmt.Errorf("let* bindings must come in pairs; found %d", len(bindings))
				}

				letEnv := environment.NewEnv(env)
				for i := 0; i < len(bindings); i += 2 {
					if bindings[i].Symbol == nil {
						return nil, fmt.Errorf("left-hand binding must be a symbol")
					}

					sym := *bindings[i].Symbol
					value := bindings[i + 1]
					evald, err := Eval(value, letEnv)
					if err != nil {
						return nil, err
					}

					letEnv.Set(sym, evald)
				}

				return Eval(list[2], letEnv)
			}
		}

		evald, err := eval_ast(ast, env)
		if err != nil {
			return nil, err
		}

		elist := *evald.List
		if elist[0] == nil || elist[0].Native == nil {
			return nil, fmt.Errorf("cannot call non-function %v", elist[0])
		}
		return elist[0].Native(elist[1:])
	}

	return eval_ast(ast, env)
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

func repl_plus(args []*types.Data) (*types.Data, error) {
	n := *args[0].Number + *args[1].Number
	return &types.Data{Number: &n}, nil
}

func repl_minus(args []*types.Data) (*types.Data, error) {
	n := *args[0].Number - *args[1].Number
	return &types.Data{Number: &n}, nil
}

func repl_times(args []*types.Data) (*types.Data, error) {
	n := *args[0].Number * *args[1].Number
	return &types.Data{Number: &n}, nil
}

func repl_div(args []*types.Data) (*types.Data, error) {
	n := *args[0].Number / *args[1].Number
	return &types.Data{Number: &n}, nil
}

func eval_ast(ast *types.Data, env *environment.Env) (*types.Data, error) {
	if ast.Symbol != nil {
		return env.Get(*ast.Symbol)
	}

	if ast.List != nil {
		ret := []*types.Data{}
		for _, expr := range *ast.List {
			evald, err := Eval(expr, env)
			if err != nil {
				return nil, err
			}

			ret = append(ret, evald)
		}
		return &types.Data{List: &ret}, nil
	}

	return ast, nil
}

var repl_env *environment.Env

func main() {
	repl_env = environment.NewEnv(nil)
	repl_env.Set("+", &types.Data{Native: repl_plus})
	repl_env.Set("-", &types.Data{Native: repl_minus})
	repl_env.Set("*", &types.Data{Native: repl_times})
	repl_env.Set("/", &types.Data{Native: repl_div})

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
