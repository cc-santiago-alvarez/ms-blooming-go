package shared_application_test

import (
	"context"
	"testing"

	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_application "github.com/futurapps/ms-blooming-go/internal/core/shared/application"
)

// contextWith devuelve un context con la sesión dada, o sin sesión si es nil.
func contextWith(s *session_domain.UserSession) context.Context {
	if s == nil {
		return context.Background()
	}
	return session_domain.SetUserSession(context.Background(), s)
}

// La agencia principal no filtra por agencia; cualquier otra filtra por su agencyId. CA-00.8
func TestAgencyScope(t *testing.T) {
	t.Run("agencia principal", func(t *testing.T) {
		ctx := contextWith(&session_domain.UserSession{
			AgencyID: "agency-main",
			Agency:   session_domain.SessionAgency{ID: "agency-main", IsMain: true},
		})
		if got := shared_application.AgencyScope(ctx); got != nil {
			t.Fatalf("AgencyScope() = %q, se esperaba nil", *got)
		}
	})

	t.Run("agencia no principal", func(t *testing.T) {
		ctx := contextWith(&session_domain.UserSession{
			AgencyID: "agency-1",
			Agency:   session_domain.SessionAgency{ID: "agency-1", IsMain: false},
		})
		got := shared_application.AgencyScope(ctx)
		if got == nil || *got != "agency-1" {
			t.Fatalf("AgencyScope() = %v, se esperaba agency-1", got)
		}
	})

	t.Run("sin sesión filtra por una agencia vacía", func(t *testing.T) {
		got := shared_application.AgencyScope(context.Background())
		if got == nil || *got != "" {
			t.Fatalf("AgencyScope() = %v, se esperaba un filtro vacío", got)
		}
	})
}

// Solo el dueño del recurso o un admin de Blooming pasan; sin sesión nadie pasa. CA-00.34
func TestIsOwnerOrAdmin(t *testing.T) {
	tests := []struct {
		name    string
		session *session_domain.UserSession
		owner   string
		want    bool
	}{
		{name: "dueño", session: &session_domain.UserSession{UserID: "user-1"}, owner: "user-1", want: true},
		{name: "otro usuario sin admin", session: &session_domain.UserSession{UserID: "user-2"}, owner: "user-1", want: false},
		{name: "admin sobre un recurso ajeno", session: &session_domain.UserSession{UserID: "admin-1", AdminBlooming: true}, owner: "user-1", want: true},
		{name: "admin con dueño vacío", session: &session_domain.UserSession{UserID: "admin-1", AdminBlooming: true}, owner: "", want: true},
		{name: "sin admin con dueño vacío", session: &session_domain.UserSession{UserID: "user-1"}, owner: "", want: false},
		{name: "dueño vacío y sesión sin userId", session: &session_domain.UserSession{}, owner: "", want: false},
		{name: "sin sesión", session: nil, owner: "user-1", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shared_application.IsOwnerOrAdmin(contextWith(tt.session), tt.owner); got != tt.want {
				t.Fatalf("IsOwnerOrAdmin(%q) = %v, se esperaba %v", tt.owner, got, tt.want)
			}
		})
	}
}
