package printer

import "fmt"
import "strconv"
import "strings"
import "types"

func PrintStr(d *types.Data, readable bool) string {
	if d.List != nil {
		outs := []string{}
		for _, m := range *d.List {
			outs = append(outs, PrintStr(m, readable))
		}
		return "(" + strings.Join(outs, " ") + ")"
	}

	if d.String != nil {
		s := *d.String
		if readable {
			s = strings.Replace(s, "\\", "\\\\", -1)
			s = strings.Replace(s, "\n", "\\n", -1)
			s = strings.Replace(s, "\"", "\\\"", -1)
			return "\"" + s + "\""
		}
		return s
	}

	if d.Number != nil {
		return strconv.Itoa(*d.Number)
	}

	if d.Symbol != nil {
		return *d.Symbol
	}

	if d.Native != nil {
		return "<native function>"
	}

	if d.Closure != nil {
		return "#<function>"
	}

	if d.Special != 0 {
		if d == types.Nil {
			return "nil"
		} else if d == types.True {
			return "true"
		} else if d == types.False {
			return "false"
		}
	}

	panic(fmt.Sprintf("Unknown Data type: %v", d))
}

