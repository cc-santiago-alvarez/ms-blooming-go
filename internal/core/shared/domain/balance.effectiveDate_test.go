package shared_domain_test

import (
	"testing"
	"time"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// EffectiveDate da la fecha del legado según el día local; desde el 26 de diciembre pasa a enero
// del año siguiente. CA-00.17
func TestEffectiveDate(t *testing.T) {
	// Salida de la lógica legacy para cada día; en diciembre desde el 26 el legado no suma el año.
	byDay := []struct {
		name string
		day  string
		want string
	}{
		{name: "5 de marzo", day: "2026-03-05", want: "2026-3-15"},
		{name: "11 de marzo", day: "2026-03-11", want: "2026-3-31"},
		{name: "25 de febrero", day: "2026-02-25", want: "2026-2-28"},
		{name: "26 de marzo", day: "2026-03-26", want: "2026-4-15"},
		{name: "28 de diciembre con el año corregido", day: "2026-12-28", want: "2027-1-15"},
		{name: "día 1", day: "2026-03-01", want: "2026-3-15"},
		{name: "día 10", day: "2026-03-10", want: "2026-3-15"},
		{name: "día 25", day: "2026-03-25", want: "2026-3-31"},
		{name: "día 31", day: "2026-03-31", want: "2026-4-15"},
		{name: "febrero bisiesto", day: "2028-02-15", want: "2028-2-29"},
		{name: "30 de abril", day: "2026-04-30", want: "2026-5-15"},
		{name: "26 de enero", day: "2026-01-26", want: "2026-2-15"},
		{name: "26 de noviembre", day: "2026-11-26", want: "2026-12-15"},
		{name: "10 de diciembre", day: "2026-12-10", want: "2026-12-15"},
		{name: "25 de diciembre", day: "2026-12-25", want: "2026-12-31"},
		{name: "31 de diciembre con el año corregido", day: "2026-12-31", want: "2027-1-15"},
	}
	for _, tt := range byDay {
		t.Run(tt.name, func(t *testing.T) {
			day, err := time.Parse(time.DateOnly, tt.day)
			if err != nil {
				t.Fatalf("fecha de prueba inválida %q: %v", tt.day, err)
			}
			// mediodía en UTC-5
			now := day.Add(17 * time.Hour)
			if got := shared_domain.EffectiveDate(now, -5); got != tt.want {
				t.Fatalf("EffectiveDate(%s, -5) = %q, se esperaba %q", tt.day, got, tt.want)
			}
		})
	}

	byZone := []struct {
		name    string
		at      string
		utcDiff int
		want    string
	}{
		{name: "día 10 en UTC-5 aunque en UTC sea 11", at: "2026-03-11T03:00:00Z", utcDiff: -5, want: "2026-3-15"},
		{name: "día 11 en UTC", at: "2026-03-11T03:00:00Z", utcDiff: 0, want: "2026-3-31"},
		{name: "31 de diciembre en UTC-5 aunque en UTC sea enero", at: "2027-01-01T02:00:00Z", utcDiff: -5, want: "2027-1-15"},
		{name: "día 26 en UTC+2 aunque en UTC sea 25", at: "2026-03-25T23:00:00Z", utcDiff: 2, want: "2026-4-15"},
	}
	for _, tt := range byZone {
		t.Run(tt.name, func(t *testing.T) {
			if got := shared_domain.EffectiveDate(mustParse(t, tt.at), tt.utcDiff); got != tt.want {
				t.Fatalf("EffectiveDate(%s, %d) = %q, se esperaba %q", tt.at, tt.utcDiff, got, tt.want)
			}
		})
	}
}
