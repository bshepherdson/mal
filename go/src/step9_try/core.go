package main

import "fmt"
import "io/ioutil"
import "strings"

import "printer"
import "reader"
import . "types"

var ns = map[string]func(list []*Data) *Data{
	"+": plus,
	"-": minus,
	"*": times,
	"/": div,

	// Atoms
	"atom":   atom,
	"atom?":  atomQ,
	"deref":  deref,
	"reset!": atomReset,
	"swap!":  atomSwap,

	// Input
	"read-string": readString,
	"slurp":       slurp,
	"eval":        eval,

	// Output
	"pr-str":  prStr,
	"str":     fStr,
	"prn":     prn,
	"println": println,

	// Lists
	"list":   mkList,
	"list?":  listQ,
	"empty?": emptyQ,
	"count":  count,
	"cons":   cons,
	"concat": concat,
	"nth":    nth,
	"first":  first,
	"rest":   rest,

	// Comparisons
	"=":  equal,
	"<":  lt,
	"<=": lte,
	">":  gt,
	">=": gte,
}

func plus(args []*Data) *Data {
	n := *args[0].Number + *args[1].Number
	return &Data{Number: &n}
}

func minus(args []*Data) *Data {
	n := *args[0].Number - *args[1].Number
	return &Data{Number: &n}
}

func times(args []*Data) *Data {
	n := *args[0].Number * *args[1].Number
	return &Data{Number: &n}
}

func div(args []*Data) *Data {
	n := *args[0].Number / *args[1].Number
	return &Data{Number: &n}
}

// Output
func printList(args []*Data, readable bool, sep string) string {
	strs := []string{}
	for _, expr := range args {
		strs = append(strs, printer.PrintStr(expr, readable))
	}

	return strings.Join(strs, sep)
}

func prStr(args []*Data) *Data {
	s := printList(args, true, " ")
	return &Data{String: &s}
}

func fStr(args []*Data) *Data {
	s := printList(args, false, "")
	return &Data{String: &s}
}

func prn(args []*Data) *Data {
	fmt.Println(printList(args, true, " "))
	return Nil
}

func println(args []*Data) *Data {
	fmt.Println(printList(args, false, " "))
	return Nil
}

// Lists
func mkList(args []*Data) *Data {
	return &Data{List: &args}
}

func listQ(args []*Data) *Data {
	if len(args) >= 1 {
		if args[0].List != nil {
			return True
		}
	}
	return False
}

func emptyQ(args []*Data) *Data {
	if len(args) == 0 || args[0].List == nil {
		return Throw(str("empty? expects a list"))
	}

	if len(*args[0].List) != 0 {
		return False
	}
	return True
}

func count(args []*Data) *Data {
	if len(args) == 0 || (args[0].List == nil && args[0] != Nil) {
		return Throw(str("count expects a list"))
	}

	num := 0
	if args[0].List != nil {
		num = len(*args[0].List)
	}
	return &Data{Number: &num}
}

func cons(args []*Data) *Data {
	if len(args) != 2 {
		return Throw(str("cons expects two arguments"))
	}

	if args[1].List == nil {
		return Throw(str("second argument to cons must be a list"))
	}

	list := []*Data{args[0]}
	list = append(list, *args[1].List...)
	return &Data{List: &list}
}

func concat(args []*Data) *Data {
	out := []*Data{}
	for _, a := range args {
		if a.List == nil {
			return Throw(str("concat expects all args to be lists"))
		}
		out = append(out, *a.List...)
	}
	return &Data{List: &out}
}

// Comparisons
func equal(args []*Data) *Data {
	if len(args) != 2 {
		return Throw(str("= expects exactly 2 arguments"))
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
		return False // This isn't ideal, but it should suffice.
	}

	if x.List != nil && y.List != nil {
		if len(*x.List) != len(*y.List) {
			return False
		}
		for i, xv := range *x.List {
			eq := equal([]*Data{xv, (*y.List)[i]})
			if HasError() {
				return nil
			}
			if eq == False {
				return eq
			}
		}
		return True
	}

	return False // Type mismatch
}

func retBool(b bool) *Data {
	if b {
		return True
	}
	return False
}

// Expects two Number arguments; fails otherwise.
func prepNumbers(args []*Data, op string) (int, int) {
	if len(args) != 2 {
		Throw(str("expected 2 args to %s, got %d", op, len(args)))
		return 0, 0
	}

	if args[0].Number == nil || args[1].Number == nil {
		Throw(str("arguments to %s must be numbers", op))
		return 0, 0
	}

	x := *args[0].Number
	y := *args[1].Number
	return x, y
}

func lt(args []*Data) *Data {
	x, y := prepNumbers(args, "<")
	if HasError() {
		return nil
	}
	return retBool(x < y)
}

func gt(args []*Data) *Data {
	x, y := prepNumbers(args, ">")
	if HasError() {
		return nil
	}
	return retBool(x > y)
}

func lte(args []*Data) *Data {
	x, y := prepNumbers(args, "<=")
	if HasError() {
		return nil
	}
	return retBool(x <= y)
}

func gte(args []*Data) *Data {
	x, y := prepNumbers(args, ">=")
	if HasError() {
		return nil
	}
	return retBool(x >= y)
}

func readString(args []*Data) *Data {
	if len(args) == 0 || args[0].String == nil {
		return Throw(str("read-string expects a single string arg"))
	}
	return reader.ReadStr(*args[0].String)
}

func slurp(args []*Data) *Data {
	if len(args) == 0 || args[0].String == nil {
		return Throw(str("slurp expects a single filename as a string"))
	}

	filename := *args[0].String
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return Throw(str("slurp failed to read the file: %v", err))
	}
	str := string(contents)
	return &Data{String: &str}
}

func eval(args []*Data) *Data {
	if len(args) != 1 {
		return Throw(str("eval expects a single value as an argument"))
	}

	return Eval(args[0], repl_env)
}

func atom(args []*Data) *Data {
	if len(args) != 1 {
		return Throw(str("atom expects a single value"))
	}
	return &Data{Atom: args[0]}
}

func atomQ(args []*Data) *Data {
	if len(args) != 1 {
		return Throw(str("atom? expects a single value"))
	}
	return retBool(args[0].Atom != nil)
}

func deref(args []*Data) *Data {
	if len(args) != 1 {
		return Throw(str("deref? expects a single value"))
	}
	if args[0].Atom == nil {
		return Throw(str("deref expects an atom"))
	}
	return args[0].Atom
}

func atomReset(args []*Data) *Data {
	if len(args) != 2 {
		return Throw(str("reset! requires two values"))
	}
	if args[0].Atom == nil {
		return Throw(str("reset! must have an atom as its first argument"))
	}

	args[0].Atom = args[1]
	return args[1]
}

func atomSwap(args []*Data) *Data {
	if len(args) < 2 {
		return Throw(str("swap! requires at least two values"))
	}
	if args[0].Atom == nil {
		return Throw(str("swap! must have an atom as its first argument"))
	}
	if args[1].Native == nil && args[1].Closure == nil {
		return Throw(str("swap! must have a function as its second argument"))
	}

	list := []*Data{args[1], args[0].Atom}
	list = append(list, args[2:]...)
	val := Eval(&Data{List: &list}, repl_env)
	if HasError() {
		return nil
	}
	args[0].Atom = val
	return val
}

func nth(args []*Data) *Data {
	if len(args) != 2 || args[0].List == nil || args[1].Number == nil {
		return Throw(str("nth expects a list and number"))
	}
	idx := *args[1].Number
	list := *args[0].List
	if idx >= len(list) {
		return Throw(str("nth: index out of bounds"))
	}
	return list[idx]
}

func first(args []*Data) *Data {
	if len(args) != 1 {
		return Throw(str("first expects a list"))
	}

	if args[0] == Nil {
		return Nil
	}
	if args[0].List == nil {
		return Throw(str("first expects a list"))
	}

	list := *args[0].List
	if len(list) == 0 {
		return Nil
	}
	return list[0]
}

func rest(args []*Data) *Data {
	if len(args) != 1 || args[0].List == nil {
		return Throw(str("rest expects a list"))
	}
	l := *args[0].List
	if len(l) == 0 {
		return args[0] // Empty lists are returned.
	}
	return list(l[1:]...)
}

var nsMal = []string{
	"(def! not (fn* (a) (if a false true)))",
	"(def! load-file (fn* (f) (eval (read-string (str \"(do \" (slurp f) \")\")))))",
	"(def! *ARGV* (list))",
	"(defmacro! cond (fn* (& xs) (if (> (count xs) 0) (list 'if (first xs) (if (> (count xs) 1) (nth xs 1) (throw \"odd number of forms to cond\")) (cons 'cond (rest (rest xs)))))))",
	"(defmacro! or (fn* (& xs) (if (empty? xs) nil (if (= 1 (count xs)) (first xs) `(let* (or_inner ~(first xs)) (if or_inner or_inner (or ~@(rest xs))))))))",
}
