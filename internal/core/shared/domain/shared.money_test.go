package shared_domain_test

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Round2 da lo mismo que Number(v.toFixed(2)) del legado y nunca devuelve -0. CA-00.37
func TestRound2(t *testing.T) {
	// pares [v, Number(v.toFixed(2))] calculados con Node
	raw, err := os.ReadFile("testdata/legacy-round2.json")
	if err != nil {
		t.Fatalf("no se pudo leer la tabla del legado: %v", err)
	}
	var pairs [][2]float64
	if err := json.Unmarshal(raw, &pairs); err != nil {
		t.Fatalf("tabla del legado inválida: %v", err)
	}
	if len(pairs) < 1000 {
		t.Fatalf("la tabla del legado tiene %d casos, se esperaban al menos 1000", len(pairs))
	}

	mismatches := 0
	for _, p := range pairs {
		if got := shared_domain.Round2(p[0]); got != p[1] {
			mismatches++
			if mismatches <= 10 {
				t.Errorf("Round2(%v) = %v, toFixed(2) da %v", p[0], got, p[1])
			}
		}
	}
	if mismatches > 0 {
		t.Errorf("%d de %d casos difieren de toFixed(2)", mismatches, len(pairs))
	}

	t.Run("sin -0", func(t *testing.T) {
		for _, v := range []float64{-0.001, -0.004999, math.Copysign(0, -1)} {
			if got := shared_domain.Round2(v); got != 0 || math.Signbit(got) {
				t.Errorf("Round2(%v) = %v, se esperaba 0 sin signo", v, got)
			}
		}
	})

	t.Run("valores no finitos", func(t *testing.T) {
		if got := shared_domain.Round2(math.NaN()); !math.IsNaN(got) {
			t.Errorf("Round2(NaN) = %v", got)
		}
		for _, v := range []float64{math.Inf(1), math.Inf(-1)} {
			if got := shared_domain.Round2(v); got != v {
				t.Errorf("Round2(%v) = %v", v, got)
			}
		}
	})
}
