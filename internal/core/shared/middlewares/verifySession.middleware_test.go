package middlewares_adapters_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	oauth_adapters "github.com/futurapps/ms-blooming-go/internal/core/shared/adapters/oauth"
)

// Con x-secret válido y sin x-token responde 401 sin llamar a ms-oauth; los endpoints internos no
// exigen x-token. CA-00.3
func TestVerifySessionRequiresToken(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
	}{
		{name: "sin x-token", headers: map[string]string{"x-secret": testAPISecret}},
		{name: "x-token en blanco", headers: map[string]string{"x-secret": testAPISecret, "x-token": "   "}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifier := &fakeSessionVerifier{session: &session_domain.UserSession{UserID: "user-1"}}
			api, p := newTestAPI(testAPISecret, verifier)

			rec := serve(api, http.MethodGet, bondsPath, tt.headers, "")

			assertError(t, rec, http.StatusUnauthorized, "TOKEN_IS_REQUIRED")
			if verifier.calls.Load() != 0 || p.calls != 0 {
				t.Fatalf("se llamó %d veces a ms-oauth y %d al handler, se esperaba 0 y 0", verifier.calls.Load(), p.calls)
			}
		})
	}

	t.Run("endpoint interno sin x-token", func(t *testing.T) {
		verifier := &fakeSessionVerifier{session: &session_domain.UserSession{UserID: "user-1"}}
		api, p := newTestAPI(testAPISecret, verifier)

		rec := serve(api, http.MethodPost, internalPath, map[string]string{"x-secret": testAPISecret}, "")

		if rec.Code != http.StatusOK || p.calls != 1 {
			t.Fatalf("status %d y %d llamadas al handler, se esperaba 200 y 1", rec.Code, p.calls)
		}
		if verifier.calls.Load() != 0 {
			t.Fatalf("se llamó %d veces a ms-oauth, se esperaba 0", verifier.calls.Load())
		}
		if p.session != nil {
			t.Fatalf("el endpoint interno recibió una sesión: %+v", p.session)
		}
	})
}

// Si ms-oauth rechaza el token, responde algo inválido o no está disponible, el ms responde 401
// TOKEN_INVALID. CA-00.4
func TestVerifySessionRejectsInvalidToken(t *testing.T) {
	tests := []struct {
		name  string
		oauth http.HandlerFunc // nil: ms-oauth caído
	}{
		{name: "ms-oauth responde 403", oauth: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}},
		{name: "ms-oauth responde 401", oauth: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}},
		{name: "ms-oauth responde 500", oauth: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}},
		{name: "JSON inválido", oauth: func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"userId": `))
		}},
		{name: "sesión sin userId", oauth: func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"agencyId": "agency-1"}`))
		}},
		{name: "ms-oauth caído"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var baseURL string
			if tt.oauth != nil {
				baseURL = newFakeOAuth(t, tt.oauth)
			} else {
				srv := httptest.NewServer(http.NotFoundHandler())
				srv.Close()
				baseURL = srv.URL + "/api/v1"
			}
			api, p := newTestAPI(testAPISecret, oauth_adapters.NewSessionClient(baseURL, testOAuthSecret))

			rec := serve(api, http.MethodGet, bondsPath, authHeaders("tok-rechazado"), "")

			assertError(t, rec, http.StatusUnauthorized, "TOKEN_INVALID")
			if p.calls != 0 {
				t.Fatalf("se llamó %d veces al handler con el token rechazado", p.calls)
			}
		})
	}
}

// Si ms-oauth no responde en 3 s, el ms responde 401 TOKEN_INVALID sin esperar al timeout de 4 s
// del cliente. CA-00.4
func TestVerifySessionOAuthTimeout(t *testing.T) {
	t.Parallel()

	baseURL := newFakeOAuth(t, func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(10 * time.Second):
		}
	})
	api, p := newTestAPI(testAPISecret, oauth_adapters.NewSessionClient(baseURL, testOAuthSecret))

	start := time.Now()
	rec := serve(api, http.MethodGet, bondsPath, authHeaders("tok-1"), "")
	elapsed := time.Since(start)

	assertError(t, rec, http.StatusUnauthorized, "TOKEN_INVALID")
	if elapsed < 3*time.Second || elapsed >= 4*time.Second {
		t.Fatalf("respondió en %s, se esperaba entre 3 s y 4 s", elapsed)
	}
	if p.calls != 0 {
		t.Fatalf("se llamó %d veces al handler sin sesión", p.calls)
	}
}

// oauthRequest es lo que el ms-oauth falso recibió.
type oauthRequest struct {
	method, path, token, secret string
}

// Con un token válido, el handler lee del context la sesión que devolvió ms-oauth. CA-00.5
func TestVerifySessionStoresOAuthSession(t *testing.T) {
	received := make(chan oauthRequest, 1)
	baseURL := newFakeOAuth(t, func(w http.ResponseWriter, r *http.Request) {
		received <- oauthRequest{
			method: r.Method,
			path:   r.URL.Path,
			token:  r.Header.Get("x-token"),
			secret: r.Header.Get("x-secret"),
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"_id": "user-1",
			"userId": "user-1",
			"token": "tok-valido",
			"agency": {"_id": "agency-1", "name": "Agencia Uno", "isMain": true},
			"agencyId": "agency-1",
			"countryCode": "MX",
			"first_name": "Ana",
			"email": "ana@futurapps.com",
			"adminBlooming": true,
			"symbolCurrency": "$"
		}`))
	})
	api, p := newTestAPI(testAPISecret, oauth_adapters.NewSessionClient(baseURL, testOAuthSecret))

	rec := serve(api, http.MethodGet, bondsPath, authHeaders("tok-valido"), "")

	if rec.Code != http.StatusOK || p.calls != 1 {
		t.Fatalf("status %d y %d llamadas al handler, se esperaba 200 y 1 (body %s)", rec.Code, p.calls, rec.Body)
	}

	got := <-received
	want := oauthRequest{method: http.MethodGet, path: "/api/v1/session", token: "tok-valido", secret: testOAuthSecret}
	if got != want {
		t.Fatalf("ms-oauth recibió %+v, se esperaba %+v", got, want)
	}

	s := p.session
	if s == nil {
		t.Fatal("el handler no recibió la sesión")
	}
	if s.UserID != "user-1" || s.AgencyID != "agency-1" || !s.Agency.IsMain || s.CountryCode != "MX" || !s.AdminBlooming {
		t.Fatalf("sesión en el context: %+v", s)
	}
}
