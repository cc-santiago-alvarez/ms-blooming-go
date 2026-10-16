package shared_domain_test

import (
	"testing"
	"time"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// mustParse lee un instante en RFC 3339.
func mustParse(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("fecha de prueba inválida %q: %v", s, err)
	}
	return v
}

// StartOfDay da el inicio del día en la zona utcDiff, expresado en UTC. CA-00.12
func TestStartOfDay(t *testing.T) {
	tests := []struct {
		name    string
		at      string
		utcDiff int
		want    string
	}{
		{name: "3 a. m. UTC es el día anterior en UTC-5", at: "2026-03-10T03:00:00Z", utcDiff: -5, want: "2026-03-09T05:00:00Z"},
		{name: "UTC", at: "2026-03-10T03:00:00Z", utcDiff: 0, want: "2026-03-10T00:00:00Z"},
		{name: "UTC+14 ya es el día siguiente", at: "2026-03-10T11:00:00Z", utcDiff: 14, want: "2026-03-10T10:00:00Z"},
		{name: "UTC-12 sigue en el día anterior", at: "2026-03-10T11:00:00Z", utcDiff: -12, want: "2026-03-09T12:00:00Z"},
		{name: "medianoche local exacta", at: "2026-03-10T05:00:00Z", utcDiff: -5, want: "2026-03-10T05:00:00Z"},
		{name: "un segundo antes de la medianoche local", at: "2026-03-10T04:59:59Z", utcDiff: -5, want: "2026-03-09T05:00:00Z"},
		{name: "cambio de año", at: "2027-01-01T02:00:00Z", utcDiff: -5, want: "2026-12-31T05:00:00Z"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shared_domain.StartOfDay(mustParse(t, tt.at), tt.utcDiff)
			if !got.Equal(mustParse(t, tt.want)) || got.Location() != time.UTC {
				t.Fatalf("StartOfDay(%s, %d) = %v, se esperaba %s en UTC", tt.at, tt.utcDiff, got, tt.want)
			}
		})
	}
}

// DaysBetween cuenta días calendario en la zona utcDiff, no bloques de 24 horas. CA-00.12
func TestDaysBetween(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		utcDiff int
		want    int
	}{
		{name: "3 horas que cruzan la medianoche local", from: "2026-03-10T03:00:00Z", to: "2026-03-10T06:00:00Z", utcDiff: -5, want: 1},
		{name: "las mismas 3 horas en UTC", from: "2026-03-10T03:00:00Z", to: "2026-03-10T06:00:00Z", utcDiff: 0, want: 0},
		{name: "23 horas dentro del mismo día local", from: "2026-03-10T05:00:00Z", to: "2026-03-11T04:00:00Z", utcDiff: -5, want: 0},
		{name: "orden inverso", from: "2026-03-10T06:00:00Z", to: "2026-03-10T03:00:00Z", utcDiff: -5, want: -1},
		{name: "cambio de mes", from: "2026-01-31T12:00:00Z", to: "2026-02-01T12:00:00Z", utcDiff: -5, want: 1},
		{name: "cambio de año", from: "2026-12-31T12:00:00Z", to: "2027-01-01T12:00:00Z", utcDiff: -5, want: 1},
		{name: "febrero bisiesto", from: "2028-02-28T12:00:00Z", to: "2028-03-01T12:00:00Z", utcDiff: -5, want: 2},
		{name: "30 días", from: "2026-03-01T15:00:00Z", to: "2026-03-31T15:00:00Z", utcDiff: -5, want: 30},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shared_domain.DaysBetween(mustParse(t, tt.from), mustParse(t, tt.to), tt.utcDiff)
			if got != tt.want {
				t.Fatalf("DaysBetween(%s, %s, %d) = %d, se esperaba %d", tt.from, tt.to, tt.utcDiff, got, tt.want)
			}
		})
	}
}
