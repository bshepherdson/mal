package main

import "fmt"
import "strings"

import "printer"
import "types"

var ns = map[string]func(list []*types.Data) (*types.Data, error){
	"+": plus,
	"-": minus,
	"*": times,
	"/": div,

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

var nsMal = []string{
	"(def! not (fn* (a) (if a false true)))",
}
