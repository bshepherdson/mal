package types

type Data interface {
}

type DList struct {
	Members []Data
}

type DString struct {
	Str string
}

type DNumber struct {
	Num int
}

type DSymbol struct {
	Name string
}
