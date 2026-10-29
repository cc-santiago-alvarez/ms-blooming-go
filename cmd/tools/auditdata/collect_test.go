package main

import (
	"slices"
	"testing"
)

// Los errores de decodificación no muestran los valores guardados que citan algunos decoders.
func TestRedactQuoted(t *testing.T) {
	cases := map[string]string{
		`bondBound: error decoding key amount: FlexNumber: "1.500,00" no es un número`: `bondBound: error decoding key amount: FlexNumber: "…" no es un número`,
		`error decoding key isRemove: cannot decode string into a boolean type`:        `error decoding key isRemove: cannot decode string into a boolean type`,
	}
	for in, want := range cases {
		if got := redactQuoted(in); got != want {
			t.Errorf("redactQuoted(%q) = %q, se esperaba %q", in, got, want)
		}
	}
}

// chunks parte los ids sin perder ni repetir ninguno.
func TestChunks(t *testing.T) {
	ids := []string{"a", "b", "c", "d", "e"}
	got := chunks(ids, 2)
	if len(got) != 3 || !slices.Equal(got[2], []string{"e"}) {
		t.Errorf("chunks = %v", got)
	}
	if len(chunks(nil, 2)) != 0 {
		t.Error("sin ids no hay grupos")
	}
}
