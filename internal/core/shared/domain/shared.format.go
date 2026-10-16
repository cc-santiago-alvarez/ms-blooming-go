package shared_domain

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Format reemplaza {0}, {1}, … de tpl por args, en una sola pasada: lo insertado no se vuelve a
// reemplazar. Un {n} sin argumento queda igual y nil se escribe vacío.
func Format(tpl string, args ...any) string {
	var b strings.Builder
	b.Grow(len(tpl))
	for i := 0; i < len(tpl); i++ {
		if tpl[i] != '{' {
			b.WriteByte(tpl[i])
			continue
		}
		end := i + 1
		for end < len(tpl) && tpl[end] >= '0' && tpl[end] <= '9' {
			end++
		}
		if end == i+1 || end >= len(tpl) || tpl[end] != '}' {
			b.WriteByte('{')
			continue
		}
		n, err := strconv.Atoi(tpl[i+1 : end])
		if err != nil || n >= len(args) {
			b.WriteString(tpl[i : end+1])
		} else {
			b.WriteString(formatArg(args[n]))
		}
		i = end
	}
	return b.String()
}

func formatArg(v any) string {
	switch v := v.(type) {
	case nil:
		return ""
	case string:
		return v
	default:
		return fmt.Sprint(v)
	}
}

// FormatCurrency redondea a 2 decimales y separa los miles con coma: 1234567.5 → "1,234,567.50".
// Un valor no finito da "0.00".
func FormatCurrency(v float64) string {
	v = Round2(v)
	if math.IsNaN(v) || math.IsInf(v, 0) {
		v = 0
	}

	digits := strconv.FormatFloat(math.Abs(v), 'f', 2, 64)
	intPart, decPart := digits[:len(digits)-3], digits[len(digits)-2:]

	var b strings.Builder
	if v < 0 {
		b.WriteByte('-')
	}
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	b.WriteByte('.')
	b.WriteString(decPart)
	return b.String()
}
