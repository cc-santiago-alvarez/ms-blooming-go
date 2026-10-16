package middlewares_adapters_test

import (
	"net/http"
	"strings"
	"testing"

	oauth_adapters "github.com/futurapps/ms-blooming-go/internal/core/shared/adapters/oauth"
)

// Con x-token abc, ninguna línea del log contiene abc ni los secrets, pase o no el request por
// cada middleware; /_health no se loguea. CA-00.22
func TestRequestLogOmitsSensitiveData(t *testing.T) {
	const (
		token       = "abc"
		wrongSecret = "secreto-equivocado"
	)
	buf := captureLog(t)

	acceptingOAuth := newFakeOAuth(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"userId": "user-1", "agencyId": "agency-1"}`))
	})
	rejectingOAuth := newFakeOAuth(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	api, _ := newTestAPI(testAPISecret, oauth_adapters.NewSessionClient(acceptingOAuth, testOAuthSecret))
	rejectingAPI, _ := newTestAPI(testAPISecret, oauth_adapters.NewSessionClient(rejectingOAuth, testOAuthSecret))

	requests := []struct {
		name       string
		api        http.Handler
		method     string
		target     string
		headers    map[string]string
		body       string
		wantStatus int
	}{
		{name: "sin x-secret", api: api, method: http.MethodGet, target: bondsPath,
			headers: map[string]string{"x-token": token}, wantStatus: http.StatusUnauthorized},
		{name: "x-secret incorrecto", api: api, method: http.MethodGet, target: bondsPath,
			headers: map[string]string{"x-secret": wrongSecret, "x-token": token}, wantStatus: http.StatusUnauthorized},
		{name: "ms-oauth rechaza el token", api: rejectingAPI, method: http.MethodGet, target: bondsPath,
			headers: authHeaders(token), wantStatus: http.StatusUnauthorized},
		{name: "admin denegado", api: api, method: http.MethodGet, target: adminPath,
			headers: authHeaders(token), wantStatus: http.StatusForbidden},
		{name: "éxito con el token en el query", api: api, method: http.MethodGet, target: bondsPath + "?q=" + token,
			headers: authHeaders(token), wantStatus: http.StatusOK},
		{name: "endpoint interno con el token en el body", api: api, method: http.MethodPost, target: internalPath,
			headers: authHeaders(token), body: `{"token": "` + token + `"}`, wantStatus: http.StatusOK},
		{name: "/_health", api: api, method: http.MethodGet, target: healthPath,
			headers: authHeaders(token), wantStatus: http.StatusOK},
	}
	for _, rq := range requests {
		rec := serve(rq.api, rq.method, rq.target, rq.headers, rq.body)
		if rec.Code != rq.wantStatus {
			t.Fatalf("%s: status %d, se esperaba %d", rq.name, rec.Code, rq.wantStatus)
		}
	}

	out := buf.String()
	wantLines := []string{
		"[blooming-test] GET /api/v1/bonds 401 ",
		"[blooming-test] GET /api/v1/admin/bonds 403 ",
		"[blooming-test] GET /api/v1/bonds 200 ",
		"[blooming-test] POST /api/v1/internal/maturity 200 ",
	}
	for _, want := range wantLines {
		if !strings.Contains(out, want) {
			t.Errorf("el log no tiene la línea %q:\n%s", want, out)
		}
	}
	// Todos los requests salvo /_health dejan una línea de RequestLog.
	if got := strings.Count(out, "[blooming-test] "); got != len(requests)-1 {
		t.Errorf("RequestLog escribió %d líneas, se esperaban %d:\n%s", got, len(requests)-1, out)
	}

	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		for _, sensitive := range []string{token, testAPISecret, testOAuthSecret, wrongSecret} {
			if strings.Contains(line, sensitive) {
				t.Errorf("la línea de log %q contiene %q", line, sensitive)
			}
		}
		if strings.Contains(line, healthPath) {
			t.Errorf("se logueó /_health: %q", line)
		}
	}
}
