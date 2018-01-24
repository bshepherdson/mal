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

func Eval(form types.Data) types.Data {
	return form
}

func Print(form types.Data) string {
	return printer.PrintStr(form, true)
}

func rep(input string) string {
	form, err := Read(input)
	if err != nil {
		return fmt.Sprintf("%v", err)
	}

	return Print(Eval(form))
}

func main() {
	for {
		line, err := readline.Readline("user> ")
		if err != nil {
			break
		}
		line = strings.TrimRight(line, "\n")
		fmt.Println(rep(line))
	}
}
