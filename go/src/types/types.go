package types

type Data struct {
	List  *[]*Data
	String *string
	Symbol *string
	Number *int
	Native func(args []*Data) (*Data, error)
}
