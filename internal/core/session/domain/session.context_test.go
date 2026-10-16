package session_domain_test

import (
	"context"
	"testing"

	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
)

// La sesión guardada en el context se recupera tal cual; sin sesión no hay nada que leer. CA-00.5
func TestUserSessionContext(t *testing.T) {
	t.Run("devuelve la misma sesión que se guardó", func(t *testing.T) {
		want := &session_domain.UserSession{UserID: "user-1", AgencyID: "agency-1"}
		ctx := session_domain.SetUserSession(context.Background(), want)

		got, ok := session_domain.GetUserSession(ctx)
		if !ok || got != want {
			t.Fatalf("GetUserSession() = (%p, %v), se esperaba (%p, true)", got, ok, want)
		}
	})

	t.Run("context sin sesión", func(t *testing.T) {
		got, ok := session_domain.GetUserSession(context.Background())
		if ok || got != nil {
			t.Fatalf("GetUserSession() = (%v, %v), se esperaba (nil, false)", got, ok)
		}
	})

	t.Run("sesión nil guardada", func(t *testing.T) {
		ctx := session_domain.SetUserSession(context.Background(), nil)
		got, ok := session_domain.GetUserSession(ctx)
		if ok || got != nil {
			t.Fatalf("GetUserSession() = (%v, %v), se esperaba (nil, false)", got, ok)
		}
	})
}
