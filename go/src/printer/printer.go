package printer

import "strconv"
import "strings"
import "types"

func PrintStr(di types.Data, readable bool) string {
	switch d := di.(type) {
	case *types.DList:
		outs := []string{}
		for _, m := range d.Members {
			outs = append(outs, PrintStr(m, readable))
		}
		return "(" + strings.Join(outs, " ") + ")"

	case *types.DString:
		s := d.Str
		if readable {
			s = strings.Replace(s, "\\", "\\\\", -1)
			s = strings.Replace(s, "\n", "\\n", -1)
			s = strings.Replace(s, "\"", "\\\"", -1)
		}
		return "\"" + s + "\""

	case *types.DNumber:
		return strconv.Itoa(d.Num)

	case *types.DSymbol:
		return d.Name

	default:
		panic("Unknown Data type")
	}
}

