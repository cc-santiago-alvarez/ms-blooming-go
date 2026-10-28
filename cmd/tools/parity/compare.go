package main

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// diff es una diferencia entre el legado y Go en un campo de un caso. Where es la fecha del punto de una serie
// o la guía, y Expected, la diferencia documentada que la explica.
type diff struct {
	Case     string
	Where    string
	Field    string
	Legacy   any
	Go       any
	Expected string
}

func (d diff) String() string {
	where := ""
	if d.Where != "" {
		where = " [" + d.Where + "]"
	}
	return fmt.Sprintf("%s%s %s: legacy=%s go=%s", d.Case, where, d.Field, formatValue(d.Legacy), formatValue(d.Go))
}

// formatValue muestra un valor de una respuesta; nil es un campo ausente.
func formatValue(v any) string {
	switch t := v.(type) {
	case nil:
		return "(ausente)"
	case string:
		return strconv.Quote(t)
	default:
		return fmt.Sprint(t)
	}
}

// num devuelve el valor numérico de v: un número de JSON o un string numérico, como los toFixed(2) del legado.
func num(v any) (float64, bool) {
	switch t := v.(type) {
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int32:
		return float64(t), true
	case int64:
		return float64(t), true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return f, err == nil
	}
	return 0, false
}

// hasDecimals dice si alguno de values es un número con parte decimal.
func hasDecimals(values ...any) bool {
	for _, v := range values {
		if f, ok := num(v); ok && f != math.Trunc(f) {
			return true
		}
	}
	return false
}

// sameValue compara un valor del legado con uno de Go: los números, redondeados a 2 decimales de los dos
// lados, porque el legado manda strings de toFixed(2) o valores sin redondear; el resto, por igualdad.
func sameValue(legacy, goValue any) bool {
	l, lok := num(legacy)
	g, gok := num(goValue)
	if lok && gok {
		return shared_domain.Round2(l) == shared_domain.Round2(g)
	}
	return reflect.DeepEqual(legacy, goValue)
}

// compareFields compara fields de los dos lados. Un campo presente de un solo lado también es una diferencia;
// null cuenta como ausente.
func compareFields(caseID, where string, legacy, goValues map[string]any, fields []string) []diff {
	var diffs []diff
	for _, field := range fields {
		l, g := legacy[field], goValues[field]
		if l == nil && g == nil {
			continue
		}
		if l == nil || g == nil || !sameValue(l, g) {
			diffs = append(diffs, diff{Case: caseID, Where: where, Field: field, Legacy: l, Go: g})
		}
	}
	return diffs
}

// compareSeries compara dos series de puntos por su date: un día que falta de un lado es una diferencia, y en
// los días comunes se comparan fields.
func compareSeries(caseID string, legacy, goPoints []map[string]any, fields []string) []diff {
	goByDate := make(map[string]map[string]any, len(goPoints))
	for _, p := range goPoints {
		goByDate[fmt.Sprint(p["date"])] = p
	}

	var diffs []diff
	seen := make(map[string]bool, len(legacy))
	for _, p := range legacy {
		date := fmt.Sprint(p["date"])
		seen[date] = true
		g, ok := goByDate[date]
		if !ok {
			diffs = append(diffs, diff{Case: caseID, Where: date, Field: "date", Legacy: date})
			continue
		}
		diffs = append(diffs, compareFields(caseID, date, p, g, fields)...)
	}
	for _, p := range goPoints {
		if date := fmt.Sprint(p["date"]); !seen[date] {
			diffs = append(diffs, diff{Case: caseID, Where: date, Field: "date", Go: date})
		}
	}
	return diffs
}

// compareGuides compara las guías por posición en date, toDate y label.
func compareGuides(caseID string, legacy, goGuides []map[string]any) []diff {
	if len(legacy) != len(goGuides) {
		return []diff{{Case: caseID, Field: "guides", Legacy: len(legacy), Go: len(goGuides)}}
	}
	var diffs []diff
	for i := range legacy {
		where := "guía " + strconv.Itoa(i+1)
		diffs = append(diffs, compareFields(caseID, where, legacy[i], goGuides[i], []string{"date", "toDate", "label"})...)
	}
	return diffs
}

// onlyFields dice si todas las diferencias son en alguno de fields.
func onlyFields(diffs []diff, fields ...string) bool {
	for _, d := range diffs {
		if !slices.Contains(fields, d.Field) {
			return false
		}
	}
	return len(diffs) > 0
}
