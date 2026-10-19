package shared_domain_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// FromNow describe la diferencia como moment: un caso por cada tramo de la tabla de umbrales, hacia el
// pasado y hacia el futuro. CA-08.7
func TestFromNow(t *testing.T) {
	now := time.Date(2026, 10, 19, 15, 0, 0, 0, time.UTC)
	day := 24 * time.Hour

	tests := []struct {
		name string
		diff time.Duration
		want string
	}{
		{name: "30 segundos", diff: -30 * time.Second, want: "a few seconds ago"},
		{name: "2 días", diff: -2 * day, want: "2 days ago"},
		{name: "400 días", diff: -400 * day, want: "a year ago"},
		{name: "44 segundos", diff: -44 * time.Second, want: "a few seconds ago"},
		{name: "45 segundos", diff: -45 * time.Second, want: "a minute ago"},
		{name: "90 segundos", diff: -90 * time.Second, want: "2 minutes ago"},
		{name: "30 minutos", diff: -30 * time.Minute, want: "30 minutes ago"},
		{name: "45 minutos", diff: -45 * time.Minute, want: "an hour ago"},
		{name: "3 horas", diff: -3 * time.Hour, want: "3 hours ago"},
		{name: "22 horas", diff: -22 * time.Hour, want: "a day ago"},
		{name: "36 horas", diff: -36 * time.Hour, want: "2 days ago"},
		{name: "26 días", diff: -26 * day, want: "a month ago"},
		{name: "60 días", diff: -60 * day, want: "2 months ago"},
		{name: "320 días", diff: -320 * day, want: "a year ago"},
		{name: "548 días", diff: -548 * day, want: "2 years ago"},
		{name: "10 años", diff: -3650 * day, want: "10 years ago"},
		{name: "misma fecha", diff: 0, want: "a few seconds ago"},
		{name: "30 segundos en el futuro", diff: 30 * time.Second, want: "in a few seconds"},
		{name: "2 días en el futuro", diff: 2 * day, want: "in 2 days"},
		{name: "400 días en el futuro", diff: 400 * day, want: "in a year"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shared_domain.FromNow(now.Add(tt.diff), now); got != tt.want {
				t.Fatalf("FromNow(now%+v, now) = %q, se esperaba %q", tt.diff, got, tt.want)
			}
		})
	}
}

// FromNow da lo mismo que moment(t).from(now) del legado en cada límite de la tabla, con meses de
// calendario, y en casos al azar; la zona de las fechas no cambia el resultado. CA-08.7
func TestFromNowMatchesLegacy(t *testing.T) {
	// salida de moment 2.29.3 (la versión del legado) evaluado con Node y TZ=UTC
	raw, err := os.ReadFile("testdata/legacy-fromnow.json")
	if err != nil {
		t.Fatalf("no se pudo leer la tabla del legado: %v", err)
	}
	var cases []struct {
		T    time.Time `json:"t"`
		Now  time.Time `json:"now"`
		Want string    `json:"want"`
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("tabla del legado inválida: %v", err)
	}
	if len(cases) < 2000 {
		t.Fatalf("la tabla del legado tiene %d casos, se esperaban al menos 2000", len(cases))
	}

	mismatches := 0
	for _, c := range cases {
		if got := shared_domain.FromNow(c.T, c.Now); got != c.Want {
			mismatches++
			if mismatches <= 10 {
				t.Errorf("FromNow(%s, %s) = %q, moment da %q", c.T.Format(time.RFC3339Nano), c.Now.Format(time.RFC3339Nano), got, c.Want)
			}
		}
	}
	if mismatches > 0 {
		t.Errorf("%d de %d casos difieren de moment", mismatches, len(cases))
	}

	t.Run("fechas en otra zona", func(t *testing.T) {
		bogota := time.FixedZone("UTC-5", -5*3600)
		for _, c := range cases {
			if got := shared_domain.FromNow(c.T.In(bogota), c.Now.In(bogota)); got != c.Want {
				t.Fatalf("FromNow en UTC-5 de %s a %s = %q, moment da %q", c.T.Format(time.RFC3339Nano), c.Now.Format(time.RFC3339Nano), got, c.Want)
			}
		}
	})
}
