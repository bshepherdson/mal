package types

type Data struct {
	List  *[]*Data
	Atom  *Data
	String *string
	Symbol *string
	Special int
	Number *int
	Native func(args []*Data) (*Data, error)
	Closure *Closure
}

type Closure struct {
	Env *Env
	Params []string   // All the positional parameters.
	TailParams string // The tail parameter, after the &, if any.
	Body *Data
}

const (
	specialNil = 1
	specialTrue
	specialFalse
)


var Nil = &Data{Special: specialNil}
var True = &Data{Special: specialTrue}
var False = &Data{Special: specialFalse}

