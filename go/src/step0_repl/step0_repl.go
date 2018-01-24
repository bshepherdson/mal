package main

import (
	"fmt"
	"strings"

	"readline"
)

func Read(raw string) string {
	return raw
}

func Eval(read string) string {
	return read
}

func Print(value string) string {
	return value
}

func rep(input string) string {
	return Print(Eval(Read(input)))
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
