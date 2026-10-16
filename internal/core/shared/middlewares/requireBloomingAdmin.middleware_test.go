package middlewares_adapters_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	middlewares_adapters "github.com/futurapps/ms-blooming-go/internal/core/shared/middlewares"
)

// Un endpoint envuelto en RequireBloomingAdmin responde 403 a una sesión sin adminBlooming. CA-00.7
func TestRequireBloomingAdmin(t *testing.T) {
	tests := []struct {
		name       string
		session    *session_domain.UserSession
		wantStatus int
	}{
		{name: "adminBlooming false", session: &session_domain.UserSession{UserID: "user-1"}, wantStatus: http.StatusForbidden},
		{name: "sin sesión", session: nil, wantStatus: http.StatusForbidden},
		{name: "adminBlooming true", session: &session_domain.UserSession{UserID: "admin-1", AdminBlooming: true}, wantStatus: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &probe{}
			handler := middlewares_adapters.RequireBloomingAdmin(p.ServeHTTP)
			req := httptest.NewRequest(http.MethodGet, adminPath, nil)
			if tt.session != nil {
				req = req.WithContext(session_domain.SetUserSession(req.Context(), tt.session))
			}
			rec := httptest.NewRecorder()

			handler(rec, req)

			if tt.wantStatus == http.StatusOK {
				if rec.Code != http.StatusOK || p.calls != 1 {
					t.Fatalf("status %d y %d llamadas al handler, se esperaba 200 y 1", rec.Code, p.calls)
				}
				return
			}
			assertError(t, rec, tt.wantStatus, "ERR_FORBIDDEN")
			if p.calls != 0 {
				t.Fatalf("se llamó %d veces al handler sin adminBlooming", p.calls)
			}
		})
	}

	apiTests := []struct {
		name       string
		target     string
		admin      bool
		wantStatus int
	}{
		{name: "endpoint de administración sin adminBlooming", target: adminPath, admin: false, wantStatus: http.StatusForbidden},
		{name: "endpoint de administración con adminBlooming", target: adminPath, admin: true, wantStatus: http.StatusOK},
		{name: "endpoint de negocio sin adminBlooming", target: bondsPath, admin: false, wantStatus: http.StatusOK},
	}
	for _, tt := range apiTests {
		t.Run(tt.name, func(t *testing.T) {
			verifier := &fakeSessionVerifier{session: &session_domain.UserSession{UserID: "user-1", AdminBlooming: tt.admin}}
			api, p := newTestAPI(testAPISecret, verifier)

			rec := serve(api, http.MethodGet, tt.target, authHeaders("tok-1"), "")

			if tt.wantStatus == http.StatusOK {
				if rec.Code != http.StatusOK || p.calls != 1 {
					t.Fatalf("status %d y %d llamadas al handler, se esperaba 200 y 1", rec.Code, p.calls)
				}
				return
			}
			assertError(t, rec, tt.wantStatus, "ERR_FORBIDDEN")
			if p.calls != 0 {
				t.Fatalf("se llamó %d veces al handler sin adminBlooming", p.calls)
			}
		})
	}
}
