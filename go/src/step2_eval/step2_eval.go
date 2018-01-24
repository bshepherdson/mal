package main

import (
	"fmt"
	"strings"

	"printer"
	"reader"
	"readline"
	"types"
)

func Read(raw string) (types.Data, error) {
	return reader.ReadStr(raw)
}

func Eval(ast types.Data, env types.Env) (types.Data, error) {
	switch form := ast.(type) {
	case *types.DList:
		if len(form.Members) == 0 {
			return form, nil
		}

		evald, err := eval_ast(ast, env)
		if err != nil {
			return nil, err
		}

		list := evald.(*types.DList)
		if fun, ok := list.Members[0].(types.DNative); ok {
			return fun(list.Members[1:])
		} else {
			return nil, fmt.Errorf("cannot call non-function")
		}

	default:
		return eval_ast(ast, env)
	}
}

func Print(form types.Data) string {
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

var repl_env types.Env = types.Env{
	"+": repl_plus,
	"-": repl_minus,
	"*": repl_times,
	"/": repl_div,
}

func repl_plus(args []types.Data) (types.Data, error) {
	x := args[0].(*types.DNumber).Num
	y := args[1].(*types.DNumber).Num
	return &types.DNumber{x + y}, nil
}

func repl_minus(args []types.Data) (types.Data, error) {
	x := args[0].(*types.DNumber).Num
	y := args[1].(*types.DNumber).Num
	return &types.DNumber{x - y}, nil
}

func repl_times(args []types.Data) (types.Data, error) {
	x := args[0].(*types.DNumber).Num
	y := args[1].(*types.DNumber).Num
	return &types.DNumber{x * y}, nil
}

func repl_div(args []types.Data) (types.Data, error) {
	x := args[0].(*types.DNumber).Num
	y := args[1].(*types.DNumber).Num
	return &types.DNumber{x / y}, nil
}

func eval_ast(ast types.Data, env types.Env) (types.Data, error) {
	switch x := ast.(type) {
	case *types.DSymbol:
		if sym, ok := env[x.Name]; ok {
			return sym, nil
		}
		return nil, fmt.Errorf("unknown symbol '%s'", x.Name)

	case *types.DList:
		ret := &types.DList{}
		for _, expr := range x.Members {
			evald, err := Eval(expr, env)
			if err != nil {
				return nil, err
			}

			ret.Members = append(ret.Members, evald)
		}
		return ret, nil

	default:
		return x, nil
	}
}



func main() {
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
