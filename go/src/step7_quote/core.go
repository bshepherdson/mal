package main

import "fmt"
import "io/ioutil"
import "strings"

import "printer"
import "reader"
import "types"

var ns = map[string]func(list []*types.Data) (*types.Data, error){
	"+": plus,
	"-": minus,
	"*": times,
	"/": div,

	// Atoms
	"atom": atom,
	"atom?": atomQ,
	"deref": deref,
	"reset!": atomReset,
	"swap!": atomSwap,

	// Input
	"read-string": readString,
	"slurp": slurp,
	"eval": eval,

	// Output
	"pr-str": prStr,
	"str": str,
	"prn": prn,
	"println": println,

	// Lists
	"list": list,
	"list?": listQ,
	"empty?": emptyQ,
	"count": count,
	"cons": cons,
	"concat": concat,

	// Comparisons
	"=": equal,
	"<": lt,
	"<=": lte,
	">": gt,
	">=": gte,
}

func plus(args []*types.Data) (*types.Data, error) {
	n := *args[0].Number + *args[1].Number
	return &types.Data{Number: &n}, nil
}

func minus(args []*types.Data) (*types.Data, error) {
	n := *args[0].Number - *args[1].Number
	return &types.Data{Number: &n}, nil
}

func times(args []*types.Data) (*types.Data, error) {
	n := *args[0].Number * *args[1].Number
	return &types.Data{Number: &n}, nil
}

func div(args []*types.Data) (*types.Data, error) {
	n := *args[0].Number / *args[1].Number
	return &types.Data{Number: &n}, nil
}

// Output
func printList(args []*types.Data, readable bool, sep string) string {
	strs := []string{}
	for _, expr := range args {
		strs = append(strs, printer.PrintStr(expr, readable))
	}

	return strings.Join(strs, sep)
}

func prStr(args []*types.Data) (*types.Data, error) {
	s := printList(args, true, " ")
	return &types.Data{String: &s}, nil
}

func str(args []*types.Data) (*types.Data, error) {
	s := printList(args, false, "")
	return &types.Data{String: &s}, nil
}

func prn(args []*types.Data) (*types.Data, error) {
	fmt.Println(printList(args, true, " "))
	return types.Nil, nil
}

func println(args []*types.Data) (*types.Data, error) {
	fmt.Println(printList(args, false, " "))
	return types.Nil, nil
}


// Lists
func list(args []*types.Data) (*types.Data, error) {
	return &types.Data{List: &args}, nil
}

func listQ(args[]*types.Data) (*types.Data, error) {
	if len(args) >= 1 {
		if args[0].List != nil {
			return types.True, nil
		}
	}
	return types.False, nil
}

func emptyQ(args []*types.Data) (*types.Data, error) {
	if len(args) == 0 || args[0].List == nil {
		return nil, fmt.Errorf("empty? expects a list")
	}

	if len(*args[0].List) != 0 {
		return types.False, nil
	}
	return types.True, nil
}

func count(args []*types.Data) (*types.Data, error) {
	if len(args) == 0 || (args[0].List == nil && args[0] != types.Nil) {
		return nil, fmt.Errorf("count expects a list")
	}

	num := 0
	if args[0].List != nil {
		num = len(*args[0].List)
	}
	return &types.Data{Number: &num}, nil
}

func cons(args []*types.Data) (*types.Data, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("cons expects two arguments")
	}

	if args[1].List == nil {
		return nil, fmt.Errorf("second argument to cons must be a list")
	}

	list := []*types.Data{args[0]}
	list = append(list, *args[1].List...)
	return &types.Data{List: &list}, nil
}

func concat(args []*types.Data) (*types.Data, error) {
	out := []*types.Data{}
	for _, a := range args {
		if a.List == nil {
			return nil, fmt.Errorf("concat expects all args to be lists")
		}
		out = append(out, *a.List...)
	}
	return &types.Data{List: &out}, nil
}

// Comparisons
func equal(args []*types.Data) (*types.Data, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("= expects exactly 2 arguments")
	}

	x := args[0]
	y := args[1]

	if x.Number != nil && y.Number != nil {
		return retBool(*x.Number == *y.Number)
	}
	if x.Symbol != nil && y.Symbol != nil {
		return retBool(*x.Symbol == *y.Symbol)
	}
	if x.String != nil && y.String != nil {
		return retBool(*x.String == *y.String)
	}
	if x.Special != 0 && y.Special != 0 {
		return retBool(x.Special == y.Special)
	}
	if x.Closure != nil && y.Closure != nil {
		return retBool(x.Closure == y.Closure)
	}
	if x.Native != nil && y.Native != nil {
		return types.False, nil // This isn't ideal, but it should suffice.
	}

	if x.List != nil && y.List != nil {
		if len(*x.List) != len(*y.List) {
			return types.False, nil
		}
		for i, xv := range *x.List {
			eq, err := equal([]*types.Data{xv, (*y.List)[i]})
			if err != nil {
				return nil, err
			}
			if eq == types.False {
				return eq, nil
			}
		}
		return types.True, nil
	}

	return types.False, nil // Type mismatch
}

func retBool(b bool) (*types.Data, error) {
	if b {
		return types.True, nil
	}
	return types.False, nil
}

// Expects two Number arguments; fails otherwise.
func prepNumbers(args []*types.Data, op string) (int, int, error) {
	if len(args) != 2 {
		return 0, 0, fmt.Errorf("expected 2 args to %s, got %d", op, len(args))
	}

	if args[0].Number == nil || args[1].Number == nil {
		return 0, 0, fmt.Errorf("arguments to %s must be numbers", op)
	}

	x := *args[0].Number
	y := *args[1].Number
	return x, y, nil
}

func lt(args []*types.Data) (*types.Data, error) {
	x, y, err := prepNumbers(args, "<")
	if err != nil {
		return nil, err
	}
	return retBool(x < y)
}

func gt(args []*types.Data) (*types.Data, error) {
	x, y, err := prepNumbers(args, ">")
	if err != nil {
		return nil, err
	}
	return retBool(x > y)
}

func lte(args []*types.Data) (*types.Data, error) {
	x, y, err := prepNumbers(args, "<=")
	if err != nil {
		return nil, err
	}
	return retBool(x <= y)
}

func gte(args []*types.Data) (*types.Data, error) {
	x, y, err := prepNumbers(args, ">=")
	if err != nil {
		return nil, err
	}
	return retBool(x >= y)
}

func readString(args []*types.Data) (*types.Data, error) {
	if len(args) == 0 || args[0].String == nil {
		return nil, fmt.Errorf("read-string expects a single string arg")
	}
	return reader.ReadStr(*args[0].String)
}

func slurp(args []*types.Data) (*types.Data, error) {
	if len(args) == 0 || args[0].String == nil {
		return nil, fmt.Errorf("slrup expects a single filename as a string")
	}

	filename := *args[0].String
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	str := string(contents)
	return &types.Data{String: &str}, nil
}

func eval(args []*types.Data) (*types.Data, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("eval expects a single value as an argument")
	}

	return Eval(args[0], repl_env)
}

func atom(args []*types.Data) (*types.Data, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("atom expects a single value")
	}
	return &types.Data{Atom: args[0]}, nil
}

func atomQ(args []*types.Data) (*types.Data, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("atom? expects a single value")
	}
	return retBool(args[0].Atom != nil)
}

func deref(args []*types.Data) (*types.Data, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("deref? expects a single value")
	}
	if args[0].Atom == nil {
		return nil, fmt.Errorf("deref expects an atom")
	}
	return args[0].Atom, nil
}

func atomReset(args []*types.Data) (*types.Data, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("reset! requires two values")
	}
	if args[0].Atom == nil {
		return nil, fmt.Errorf("reset! must have an atom as its first argument")
	}

	args[0].Atom = args[1]
	return args[1], nil
}

func atomSwap(args []*types.Data) (*types.Data, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("swap! requires at least two values")
	}
	if args[0].Atom == nil {
		return nil, fmt.Errorf("swap! must have an atom as its first argument")
	}
	if args[1].Native == nil && args[1].Closure == nil {
		return nil, fmt.Errorf("swap! must have a function as its second argument")
	}

	list := []*types.Data{args[1], args[0].Atom}
	list = append(list, args[2:]...)
	val, err := Eval(&types.Data{List: &list}, repl_env)
	if err != nil {
		return nil, err
	}
	args[0].Atom = val
	return val, nil
}

var nsMal = []string{
	"(def! not (fn* (a) (if a false true)))",
	"(def! load-file (fn* (f) (eval (read-string (str \"(do \" (slurp f) \")\")))))",
	"(def! *ARGV* (list))",
}
