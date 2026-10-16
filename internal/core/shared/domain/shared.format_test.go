package shared_domain_test

import (
	"math"
	"testing"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Format reemplaza {n} por el argumento n en una sola pasada y deja igual lo que no es un
// placeholder con argumento. CA-00.21
func TestFormat(t *testing.T) {
	eleven := []any{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}

	tests := []struct {
		name string
		tpl  string
		args []any
		want string
	}{
		{name: "un argumento", tpl: "Bono {0} vendido", args: []any{"AB-3"}, want: "Bono AB-3 vendido"},
		{name: "varios argumentos", tpl: "Compra de Bono ({0}), por $ {1}", args: []any{"AB-3", "1,500.00"}, want: "Compra de Bono (AB-3), por $ 1,500.00"},
		{name: "una sola pasada", tpl: "{0} y {1}", args: []any{"{1}", "x"}, want: "{1} y x"},
		{name: "placeholder sin argumento", tpl: "Bono {0} de {1}", args: []any{"AB-3"}, want: "Bono AB-3 de {1}"},
		{name: "nil se escribe vacío", tpl: "[{0}]", args: []any{nil}, want: "[]"},
		{name: "números y booleanos", tpl: "{0} - {1} - {2}", args: []any{1500.5, 3, true}, want: "1500.5 - 3 - true"},
		{name: "índice de dos dígitos", tpl: "{10}", args: eleven, want: "k"},
		{name: "placeholder repetido", tpl: "{0}{0}", args: []any{"a"}, want: "aa"},
		{name: "llaves que no son placeholders", tpl: "{ {a} {0 {} {-1}", args: []any{"x"}, want: "{ {a} {0 {} {-1}"},
		{name: "llave al final", tpl: "Total {", args: []any{"x"}, want: "Total {"},
		{name: "sin argumentos", tpl: "Bono {0}", want: "Bono {0}"},
		{name: "texto con tildes", tpl: "Señor(a) <strong>{0}</strong>", args: []any{"Ñandú"}, want: "Señor(a) <strong>Ñandú</strong>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shared_domain.Format(tt.tpl, tt.args...); got != tt.want {
				t.Fatalf("Format(%q, %v) = %q, se esperaba %q", tt.tpl, tt.args, got, tt.want)
			}
		})
	}
}

// FormatCurrency redondea con Round2, usa coma de miles y punto decimal, y da "0.00" para un valor
// no finito. CA-00.21
func TestFormatCurrency(t *testing.T) {
	tests := []struct {
		name string
		v    float64
		want string
	}{
		{name: "millones", v: 1234567.5, want: "1,234,567.50"},
		{name: "cero", v: 0, want: "0.00"},
		{name: "centenas", v: 100, want: "100.00"},
		{name: "miles", v: 1000, want: "1,000.00"},
		{name: "decimales sin truncar", v: 1234.56, want: "1,234.56"},
		{name: "redondeo que suma una cifra", v: 999.999, want: "1,000.00"},
		{name: "negativo", v: -1234.5, want: "-1,234.50"},
		{name: "negativo que redondea a cero", v: -0.001, want: "0.00"},
		{name: "empate como toFixed", v: 2.675, want: "2.67"},
		{name: "empate pequeño como toFixed", v: 0.015, want: "0.01"},
		{name: "mil millones", v: 1e9, want: "1,000,000,000.00"},
		{name: "NaN", v: math.NaN(), want: "0.00"},
		{name: "infinito", v: math.Inf(1), want: "0.00"},
		{name: "menos infinito", v: math.Inf(-1), want: "0.00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shared_domain.FormatCurrency(tt.v); got != tt.want {
				t.Fatalf("FormatCurrency(%v) = %q, se esperaba %q", tt.v, got, tt.want)
			}
		})
	}
}
