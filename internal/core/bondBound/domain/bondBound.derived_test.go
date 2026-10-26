package bondBound_domain_test

import (
	"testing"
	"time"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
)

// at lee un instante en RFC 3339.
func at(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("fecha de prueba inválida %q: %v", s, err)
	}
	return v
}

// withStatus devuelve el bono con el estado dado.
func withStatus(b bondBound_domain.BondBound, s bondBound_domain.BondBoundStatus) bondBound_domain.BondBound {
	b.StatusID = s
	return b
}

// Accumulated prorratea capital más interés por los días calendario transcurridos en la zona de la sesión,
// solo para un bono iniciado dentro de su plazo; si no, el monto es 0. CA-06.4
func TestAccumulated(t *testing.T) {
	// acumulateAmount de bond.bound.js evaluado con Node con los mismos términos y redondeado con toFixed(2)
	example := withStatus(quoteBound(1000, 100, 10, 30, 50, 0), bondBound_domain.StatusInitiated)
	fractional := withStatus(quoteBound(1000.5, 365, 7.5, 30, 62.5, 15), bondBound_domain.StatusInitiated)
	rounding := withStatus(quoteBound(333.33, 90, 12.5, 30, 50, 0), bondBound_domain.StatusInitiated)
	noTerm := withStatus(quoteBound(1000, 0, 10, 0, 50, 0), bondBound_domain.StatusInitiated)

	tests := []struct {
		name    string
		bound   bondBound_domain.BondBound
		now     time.Time
		utcDiff int
		days    int
		amount  float64
	}{
		{name: "día 50", bound: example, now: quoteAt(50, -5), utcDiff: -5, days: 50, amount: 550},
		{name: "estado madurado", bound: withStatus(example, bondBound_domain.StatusFinalized), now: quoteAt(50, -5), utcDiff: -5, days: 50, amount: 0},
		{name: "estado bloqueado", bound: withStatus(example, bondBound_domain.StatusBlocked), now: quoteAt(50, -5), utcDiff: -5, days: 50, amount: 0},
		{name: "día 101, pasado el plazo", bound: example, now: quoteAt(101, -5), utcDiff: -5, days: 101, amount: 0},
		{name: "vencimiento", bound: example, now: quoteAt(100, -5), utcDiff: -5, days: 100, amount: 1100},
		{name: "día de la compra", bound: example, now: quoteAt(0, -5), utcDiff: -5, days: 0, amount: 0},
		{name: "día 1", bound: example, now: quoteAt(1, -5), utcDiff: -5, days: 1, amount: 11},
		{name: "tasa con decimales, día 100", bound: fractional, now: quoteAt(100, 14), utcDiff: 14, days: 100, amount: 294.67},
		{name: "tasa con decimales, víspera del vencimiento", bound: fractional, now: quoteAt(364, 14), utcDiff: 14, days: 364, amount: 1072.59},
		{name: "tasa con decimales, vencimiento", bound: fractional, now: quoteAt(365, 14), utcDiff: 14, days: 365, amount: 1075.54},
		{name: "redondeo a dos decimales", bound: rounding, now: quoteAt(7, -5), utcDiff: -5, days: 7, amount: 29.17},
		{name: "sin plazo", bound: noTerm, now: quoteAt(5, -5), utcDiff: -5, days: 5, amount: 0},
		// la compra es el 13-oct en UTC-5 y el 14-oct en UTC
		{name: "días contados en UTC-5", bound: example, now: at(t, "2026-10-14T12:00:00Z"), utcDiff: -5, days: 1, amount: 11},
		{name: "el mismo instante en UTC", bound: example, now: at(t, "2026-10-14T12:00:00Z"), utcDiff: 0, days: 0, amount: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			days, amount := bondBound_domain.Accumulated(tt.bound, tt.now, tt.utcDiff)

			if days != tt.days || amount != tt.amount {
				t.Errorf("daysPassed %d y acumulado %v, se esperaban %d y %v", days, amount, tt.days, tt.amount)
			}
		})
	}
}

// Progress marca growing desde el día de la inflexión hasta el último día del plazo y grew desde el día
// siguiente, con los días calendario contados en la zona de la sesión. CA-06.5
func TestProgress(t *testing.T) {
	bound := quoteBound(1000, 100, 10, 30, 50, 0)
	noInflection := quoteBound(1000, 100, 10, 0, 50, 0)

	tests := []struct {
		name    string
		bound   bondBound_domain.BondBound
		now     time.Time
		utcDiff int
		growing bool
		grew    bool
	}{
		{name: "día de la compra", bound: bound, now: quoteAt(0, -5), utcDiff: -5, growing: false, grew: false},
		{name: "día 29", bound: bound, now: quoteAt(29, -5), utcDiff: -5, growing: false, grew: false},
		{name: "día 30, inflexión", bound: bound, now: quoteAt(30, -5), utcDiff: -5, growing: true, grew: false},
		{name: "día 100, vencimiento", bound: bound, now: quoteAt(100, -5), utcDiff: -5, growing: true, grew: false},
		{name: "día 101", bound: bound, now: quoteAt(101, -5), utcDiff: -5, growing: false, grew: true},
		{name: "día 400", bound: bound, now: quoteAt(400, -5), utcDiff: -5, growing: false, grew: true},
		// la compra es el 13-oct en UTC-5 y el 14-oct en UTC
		{name: "día 30 en UTC-5", bound: bound, now: at(t, "2026-11-12T12:00:00Z"), utcDiff: -5, growing: true, grew: false},
		{name: "el mismo instante es el día 29 en UTC", bound: bound, now: at(t, "2026-11-12T12:00:00Z"), utcDiff: 0, growing: false, grew: false},
		{name: "inflexión en el día de la compra", bound: noInflection, now: quoteAt(0, -5), utcDiff: -5, growing: true, grew: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			growing, grew := bondBound_domain.Progress(tt.bound, tt.now, tt.utcDiff)

			if growing != tt.growing || grew != tt.grew {
				t.Errorf("growing %v y grew %v, se esperaban %v y %v", growing, grew, tt.growing, tt.grew)
			}
		})
	}
}
