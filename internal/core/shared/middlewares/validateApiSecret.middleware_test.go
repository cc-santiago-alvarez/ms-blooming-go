package middlewares_adapters_test

import (
	"net/http"
	"strings"
	"testing"

	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
)

// Sin x-secret o con un valor distinto, todo endpoint no público responde 401 sin llegar a
// ms-oauth; /_health no lo exige. CA-00.2
func TestValidateApiSecret(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		target     string
		headers    map[string]string
		wantStatus int
	}{
		{
			name: "sin x-secret", method: http.MethodGet, target: bondsPath,
			headers:    map[string]string{"x-token": "tok-1"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "x-secret distinto", method: http.MethodGet, target: bondsPath,
			headers:    map[string]string{"x-secret": "otro-valor", "x-token": "tok-1"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "x-secret con un carácter de más", method: http.MethodGet, target: bondsPath,
			headers:    map[string]string{"x-secret": testAPISecret + "x", "x-token": "tok-1"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "x-secret con un carácter de menos", method: http.MethodGet, target: bondsPath,
			headers:    map[string]string{"x-secret": testAPISecret[:len(testAPISecret)-1], "x-token": "tok-1"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "x-secret en mayúsculas", method: http.MethodGet, target: bondsPath,
			headers:    map[string]string{"x-secret": strings.ToUpper(testAPISecret), "x-token": "tok-1"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "endpoint de administración sin x-secret", method: http.MethodGet, target: adminPath,
			headers:    map[string]string{"x-token": "tok-1"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "endpoint interno sin x-secret", method: http.MethodPost, target: internalPath,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "x-secret correcto", method: http.MethodGet, target: bondsPath,
			headers:    authHeaders("tok-1"),
			wantStatus: http.StatusOK,
		},
		{
			name: "x-secret correcto con espacios y caracteres invisibles", method: http.MethodGet, target: bondsPath,
			headers:    map[string]string{"x-secret": " \t" + testAPISecret + "​", "x-token": "tok-1"},
			wantStatus: http.StatusOK,
		},
		{
			name: "endpoint interno con x-secret y sin x-token", method: http.MethodPost, target: internalPath,
			headers:    map[string]string{"x-secret": testAPISecret},
			wantStatus: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifier := &fakeSessionVerifier{session: &session_domain.UserSession{UserID: "user-1"}}
			api, p := newTestAPI(testAPISecret, verifier)

			rec := serve(api, tt.method, tt.target, tt.headers, "")

			if tt.wantStatus == http.StatusOK {
				if rec.Code != http.StatusOK || p.calls != 1 {
					t.Fatalf("status %d y %d llamadas al handler, se esperaba 200 y 1", rec.Code, p.calls)
				}
				return
			}
			assertError(t, rec, tt.wantStatus, "UNAUTHORIZED_INVALID_SECRET")
			if p.calls != 0 || verifier.calls.Load() != 0 {
				t.Fatalf("con el secret rechazado se llamó %d veces al handler y %d a ms-oauth", p.calls, verifier.calls.Load())
			}
		})
	}

	t.Run("/_health no exige x-secret", func(t *testing.T) {
		api, _ := newTestAPI(testAPISecret, &fakeSessionVerifier{})

		rec := serve(api, http.MethodGet, healthPath, nil, "")

		if rec.Code != http.StatusOK {
			t.Fatalf("status %d, se esperaba 200", rec.Code)
		}
		if got := rec.Body.String(); got != testServerName+" is running" {
			t.Fatalf("body %q, se esperaba %q", got, testServerName+" is running")
		}
	})
}

// API_SECRET se compara sin caracteres invisibles, y uno vacío rechaza todo request. CA-00.2
func TestValidateApiSecretConfiguredValue(t *testing.T) {
	tests := []struct {
		name       string
		apiSecret  string
		xSecret    string
		wantStatus int
	}{
		{name: "API_SECRET con salto de línea final", apiSecret: testAPISecret + "\r\n", xSecret: testAPISecret, wantStatus: http.StatusOK},
		{name: "API_SECRET vacío y x-secret vacío", apiSecret: "", xSecret: "", wantStatus: http.StatusUnauthorized},
		{name: "API_SECRET vacío y x-secret con valor", apiSecret: "", xSecret: "cualquiera", wantStatus: http.StatusUnauthorized},
		{name: "API_SECRET de solo espacios", apiSecret: " \t\r\n", xSecret: " ", wantStatus: http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifier := &fakeSessionVerifier{session: &session_domain.UserSession{UserID: "user-1"}}
			api, p := newTestAPI(tt.apiSecret, verifier)

			rec := serve(api, http.MethodGet, bondsPath, map[string]string{"x-secret": tt.xSecret, "x-token": "tok-1"}, "")

			if tt.wantStatus == http.StatusOK {
				if rec.Code != http.StatusOK || p.calls != 1 {
					t.Fatalf("status %d y %d llamadas al handler, se esperaba 200 y 1", rec.Code, p.calls)
				}
				return
			}
			assertError(t, rec, tt.wantStatus, "UNAUTHORIZED_INVALID_SECRET")
			if p.calls != 0 {
				t.Fatalf("con el secret rechazado se llamó %d veces al handler", p.calls)
			}
		})
	}
}
