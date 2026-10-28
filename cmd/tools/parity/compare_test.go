package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

// num lee los números de JSON y los strings numéricos del legado, y rechaza el resto.
func TestNum(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want float64
		ok   bool
	}{
		{name: "json.Number", in: json.Number("105.5"), want: 105.5, ok: true},
		{name: "toFixed", in: "105.50", want: 105.5, ok: true},
		{name: "string con espacios", in: " 12 ", want: 12, ok: true},
		{name: "int32 de Mongo", in: int32(100), want: 100, ok: true},
		{name: "float64", in: 1.25, want: 1.25, ok: true},
		{name: "texto", in: "lastBullet", ok: false},
		{name: "bool", in: true, ok: false},
		{name: "nil", in: nil, ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := num(tt.in)
			if ok != tt.ok || (ok && got != tt.want) {
				t.Errorf("num(%v) = %v, %v; se esperaba %v, %v", tt.in, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// hasDecimals detecta un número con decimales, también guardado como string.
func TestHasDecimals(t *testing.T) {
	if hasDecimals(int32(100), "10", json.Number("50"), nil) {
		t.Error("enteros, como número o string, no tienen decimales")
	}
	if !hasDecimals(int32(100), "12.5") {
		t.Error("\"12.5\" tiene decimales")
	}
}

// Los números se comparan redondeados a 2 decimales de los dos lados; el resto, por igualdad.
func TestSameValue(t *testing.T) {
	tests := []struct {
		name   string
		legacy any
		goV    any
		want   bool
	}{
		{name: "toFixed contra número", legacy: "105.50", goV: json.Number("105.5"), want: true},
		{name: "sin redondear contra redondeado", legacy: json.Number("1033.3333333"), goV: json.Number("1033.33"), want: true},
		{name: "un centavo de diferencia", legacy: "105.51", goV: json.Number("105.5"), want: false},
		{name: "bool igual", legacy: true, goV: true, want: true},
		{name: "bool distinto", legacy: true, goV: false, want: false},
		{name: "texto", legacy: "3 Oct 2026", goV: "3 Oct 2026", want: true},
		{name: "número contra texto", legacy: "lastBullet", goV: json.Number("1"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sameValue(tt.legacy, tt.goV); got != tt.want {
				t.Errorf("sameValue(%v, %v) = %v, se esperaba %v", tt.legacy, tt.goV, got, tt.want)
			}
		})
	}
}

// compareFields informa los campos distintos y los presentes de un solo lado; null cuenta como ausente.
func TestCompareFields(t *testing.T) {
	legacy := map[string]any{"profit": "50.00", "amount": json.Number("1050"), "atMoment": nil}
	goValues := map[string]any{"profit": json.Number("100"), "amount": json.Number("1050"), "totalDaysPassed": json.Number("100")}

	got := compareFields("bondBound b1", "", legacy, goValues, []string{"profit", "amount", "atMoment", "totalDaysPassed"})

	want := []diff{
		{Case: "bondBound b1", Field: "profit", Legacy: "50.00", Go: json.Number("100")},
		{Case: "bondBound b1", Field: "totalDaysPassed", Go: json.Number("100")},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("diferencias %v, se esperaba %v", got, want)
	}
}

// compareSeries cruza los puntos por date: compara los días comunes e informa los que faltan de cada lado.
func TestCompareSeries(t *testing.T) {
	legacy := []map[string]any{
		{"date": "2026-10-28", "inversion": json.Number("1000")},
		{"date": "2026-10-29", "inversion": json.Number("1000"), "ganancia": "1010.00"},
		{"date": "2026-10-30", "ganancia": "1020.00"},
	}
	goPoints := []map[string]any{
		{"date": "2026-10-28", "inversion": json.Number("1000")},
		{"date": "2026-10-29", "inversion": json.Number("1000"), "ganancia": json.Number("1010.01")},
		{"date": "2026-10-31", "ganancia": json.Number("1030")},
	}

	got := compareSeries("bondObject o1", legacy, goPoints, []string{"inversion", "ganancia"})

	want := []diff{
		{Case: "bondObject o1", Where: "2026-10-29", Field: "ganancia", Legacy: "1010.00", Go: json.Number("1010.01")},
		{Case: "bondObject o1", Where: "2026-10-30", Field: "date", Legacy: "2026-10-30"},
		{Case: "bondObject o1", Where: "2026-10-31", Field: "date", Go: "2026-10-31"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("diferencias\n%v\nse esperaba\n%v", got, want)
	}
}

// compareGuides compara las guías por posición; si no tienen la misma cantidad, informa solo eso.
func TestCompareGuides(t *testing.T) {
	legacy := []map[string]any{{"date": "2026-11-07", "label": "Punto de Crecimiento", "lineColor": "#CC0000"}}
	goGuides := []map[string]any{{"date": "2026-11-07", "label": "Punto de crecimiento", "lineColor": "#000"}}

	got := compareGuides("bondObject o1", legacy, goGuides)
	want := []diff{{Case: "bondObject o1", Where: "guía 1", Field: "label", Legacy: "Punto de Crecimiento", Go: "Punto de crecimiento"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("diferencias %v, se esperaba %v", got, want)
	}

	got = compareGuides("bondObject o1", legacy, append(goGuides, map[string]any{}))
	want = []diff{{Case: "bondObject o1", Field: "guides", Legacy: 1, Go: 2}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("con distinta cantidad de guías %v, se esperaba %v", got, want)
	}
}
