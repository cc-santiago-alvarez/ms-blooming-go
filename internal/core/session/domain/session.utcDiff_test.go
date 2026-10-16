package session_domain_test

import (
	"context"
	"encoding/json"
	"testing"

	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
)

// setDefaultUtcDiff fija DEFAULT_UTC_DIFF durante el test y restaura el valor anterior al terminar.
func setDefaultUtcDiff(t *testing.T, hours int) {
	t.Helper()
	previous := session_domain.UtcDiff(context.Background())
	session_domain.SetDefaultUtcDiff(hours)
	t.Cleanup(func() { session_domain.SetDefaultUtcDiff(previous) })
}

// contextWith devuelve un context con la sesión dada.
func contextWith(s *session_domain.UserSession) context.Context {
	return session_domain.SetUserSession(context.Background(), s)
}

// Sin utcDiff en la sesión, se usa el mapa por país y, fuera de él, DEFAULT_UTC_DIFF. CA-00.6
func TestUtcDiffByCountry(t *testing.T) {
	// Un default distinto de -5 muestra que se usa el configurado y no uno fijo.
	setDefaultUtcDiff(t, -3)

	tests := []struct {
		name    string
		country string
		want    int
	}{
		{name: "México", country: "MX", want: -7},
		{name: "Argentina, fuera del mapa", country: "AR", want: -3},
		{name: "Colombia", country: "CO", want: -5},
		{name: "Venezuela", country: "VE", want: -4},
		{name: "Estados Unidos", country: "US", want: -6},
		{name: "minúsculas y espacios", country: " mx ", want: -7},
		{name: "sin país", country: "", want: -3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := contextWith(&session_domain.UserSession{UserID: "user-1", CountryCode: tt.country})
			if got := session_domain.UtcDiff(ctx); got != tt.want {
				t.Fatalf("UtcDiff() con país %q = %d, se esperaba %d", tt.country, got, tt.want)
			}
		})
	}

	t.Run("context sin sesión", func(t *testing.T) {
		if got := session_domain.UtcDiff(context.Background()); got != -3 {
			t.Fatalf("UtcDiff() sin sesión = %d, se esperaba -3", got)
		}
	})
}

// utcDiff y gmt de la sesión ganan al país si son enteros en [-12, 14]; cualquier otro valor se
// ignora sin romper la lectura de la sesión. CA-00.6
func TestUtcDiffFromSessionFields(t *testing.T) {
	setDefaultUtcDiff(t, -3)

	tests := []struct {
		name   string
		fields string // campos que se agregan a una sesión con countryCode MX (-7)
		want   int
	}{
		{name: "utcDiff 0 es válido", fields: `"utcDiff": 0`, want: 0},
		{name: "utcDiff gana a gmt", fields: `"utcDiff": -3, "gmt": 2`, want: -3},
		{name: "solo gmt como string", fields: `"gmt": "2"`, want: 2},
		{name: "utcDiff como string", fields: `"utcDiff": "-4"`, want: -4},
		{name: "borde inferior", fields: `"utcDiff": -12`, want: -12},
		{name: "borde superior", fields: `"utcDiff": 14`, want: 14},
		{name: "utcDiff inválido usa gmt", fields: `"utcDiff": "abc", "gmt": 1`, want: 1},
		{name: "mayor a 14", fields: `"utcDiff": 15`, want: -7},
		{name: "menor a -12", fields: `"utcDiff": -13`, want: -7},
		{name: "fraccionario", fields: `"utcDiff": -4.5`, want: -7},
		{name: "string fraccionario", fields: `"gmt": "-4.5"`, want: -7},
		{name: "string no numérico", fields: `"utcDiff": "abc"`, want: -7},
		{name: "booleano", fields: `"utcDiff": true`, want: -7},
		{name: "objeto", fields: `"utcDiff": {"hours": -3}`, want: -7},
		{name: "arreglo", fields: `"utcDiff": [-3]`, want: -7},
		{name: "null", fields: `"utcDiff": null, "gmt": null`, want: -7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := `{"userId": "user-1", "countryCode": "MX", ` + tt.fields + `}`
			var s session_domain.UserSession
			if err := json.Unmarshal([]byte(payload), &s); err != nil {
				t.Fatalf("json.Unmarshal(%s) = %v, la sesión no debe fallar por utcDiff o gmt", payload, err)
			}
			if s.UserID != "user-1" || s.CountryCode != "MX" {
				t.Fatalf("la sesión perdió campos: %+v", s)
			}
			if got := session_domain.UtcDiff(contextWith(&s)); got != tt.want {
				t.Fatalf("UtcDiff() con %s = %d, se esperaba %d", tt.fields, got, tt.want)
			}
		})
	}

	t.Run("sin sesión usa el fallback", func(t *testing.T) {
		if got := session_domain.ResolveUtcDiff(nil, 9); got != 9 {
			t.Fatalf("ResolveUtcDiff(nil, 9) = %d, se esperaba 9", got)
		}
	})
}
