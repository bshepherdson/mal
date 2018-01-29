package reader

import "fmt"
import "strconv"
import . "types"

type MalReader struct {
	tokens []string
	index  int
}

func (r *MalReader) Next() (string, bool) {
	t, ok := r.Peek()
	if !ok {
		return t, false
	}

	r.index++
	return t, true
}

func (r *MalReader) Peek() (string, bool) {
	if r.index >= len(r.tokens) {
		return "EOF", false
	}
	return r.tokens[r.index], true
}

func tokenizer(input string) ([]string, error) {
	//fmt.Printf("tokenizer input: \"%s\"\n", input)
	t := make([]string, 0, 16)
	for pos := 0; pos < len(input); {
		c := input[pos]
		switch c {
		case ' ', '\r', '\n', '\t', ',':
			pos++
			continue // Whitespace and commas are skipped.

		case '~':
			if input[pos+1] == '@' { // ~@ is a thing
				t = append(t, "~@")
				pos += 2
			} else {
				t = append(t, "~") // so is just ~
				pos++
			}

		case '[', ']', '{', '}', '(', ')', '\'', '`', '^', '@':
			t = append(t, string(c))
			pos++

		case '"': // Quoted strings as one character.
			wasSlash := false
			foundEnd := false
			out := []byte{'"'}
			for end := pos + 1; end < len(input); end++ {
				if !wasSlash && input[end] == '"' {
					foundEnd = true
					out = append(out, '"')
					t = append(t, string(out))
					pos = end + 1
					break
				}

				if wasSlash {
					if input[end] == 'n' {
						out = append(out, '\n')
					} else if input[end] == '"' {
						out = append(out, '"')
					} else if input[end] == '\\' {
						out = append(out, '\\')
					} else {
						out = append(out, input[end])
					}
					wasSlash = false
				} else {
					if input[end] == '\\' {
						wasSlash = true
					} else {
						out = append(out, input[end])
					}
				}
			}

			if !foundEnd {
				return nil, fmt.Errorf("expected '\"', got EOF")
			}

		case ';': // Captures the rest of the line as a comment token.
			end := pos + 1
			for ; end < len(input); end++ {
				if input[end] == '\n' {
					pos = end
					break
				}
			}

			if end == len(input) {
				return t, nil
			}

		default:
			// Keep going until we see something special.
			end := pos + 1
		nonspec_loop:
			for end < len(input) {
				ce := input[end]
				switch ce {
				case ' ', '\t', '\n', ',', '(', ')', '[', ']', '{', '}', '~', '\'', '"', '@', '^', '`':
					//fmt.Printf("Breaking, found %c at %d\n", ce, end)
					break nonspec_loop
				}
				end++
			}
			s := input[pos:end]
			//fmt.Printf("Nonspecial, found (%d, %d) \"%s\"\n", pos, end, s)
			t = append(t, s)
			pos = end
		}
	}
	return t, nil
}

func ReadStr(input string) *Data {
	tokens, err := tokenizer(input)
	if err != nil {
		msg := fmt.Sprintf("tokenization error: %v", err)
		return Throw(&Data{String: &msg})
	}

	r := &MalReader{tokens, 0}
	return ReadForm(r)
}

func ReadForm(r *MalReader) *Data {
	t, ok := r.Peek()
	if !ok {
		msg := "expected form, got EOF"
		return Throw(&Data{String: &msg})
	}

	switch t {
	case "'":
		return nextWrapped(r, "quote")
	case "`":
		return nextWrapped(r, "quasiquote")
	case "~":
		return nextWrapped(r, "unquote")
	case "~@":
		return nextWrapped(r, "splice-unquote")
	case "(":
		return readList(r)
	default:
		return readAtom(r)
	}
}

func nextWrapped(r *MalReader, wrapper string) *Data {
	r.Next()
	next := ReadForm(r) // Read the next form.
	if HasError() {
		return nil
	}

	list := []*Data{
		&Data{Symbol: &wrapper},
		next,
	}
	return &Data{List: &list}
}

func readList(r *MalReader) *Data {
	t, ok := r.Next() // Skip the opening (
	ret := []*Data{}
	for t, ok = r.Peek(); t != ")" && ok; t, ok = r.Peek() {
		f := ReadForm(r)
		if HasError() {
			return nil
		}
		ret = append(ret, f)
	}
	if t != ")" {
		msg := "expected ')' but got EOF"
		return Throw(&Data{String: &msg})
	}

	// Found the ")"
	r.Next() // Skip over it.
	return &Data{List: &ret}
}

func readAtom(r *MalReader) *Data {
	t, ok := r.Next()
	if !ok {
		msg := "expected atom, got EOF"
		return Throw(&Data{String: &msg})
	}

	if t[0] == '"' {
		var s = t[1 : len(t)-1]
		return &Data{String: &s}
	} else if (len(t) >= 2 && t[0] == '-' && '0' <= t[1] && t[1] <= '9') || ('0' <= t[0] && t[0] <= '9') {
		n, err := strconv.Atoi(t)
		if err != nil {
			msg := "badly formatted number"
			return Throw(&Data{String: &msg})
		}
		return &Data{Number: &n}
	} else if t == "nil" {
		return Nil
	} else if t == "true" {
		return True
	} else if t == "false" {
		return False
	} else {
		return &Data{Symbol: &t}
	}
}
