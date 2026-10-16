package middlewares_adapters_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/codecraftkit/nexus"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	middlewares_adapters "github.com/futurapps/ms-blooming-go/internal/core/shared/middlewares"
	shared_ports "github.com/futurapps/ms-blooming-go/internal/core/shared/ports"
)

const (
	testServerName  = "blooming-test"
	testAPISecret   = "s3cr3t-api-valor"
	testOAuthSecret = "s3cr3t-oauth-valor"

	bondsPath    = "/api/v1/bonds"
	adminPath    = "/api/v1/admin/bonds"
	internalPath = "/api/v1/internal/maturity"
	healthPath   = "/_health"
)

// probe es el handler final: cuenta las llamadas y guarda la sesión que recibió.
type probe struct {
	calls   int
	session *session_domain.UserSession
}

func (p *probe) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.calls++
	p.session, _ = session_domain.GetUserSession(r.Context())
	w.WriteHeader(http.StatusOK)
}

// fakeSessionVerifier devuelve siempre la misma sesión o el mismo error y cuenta las llamadas.
type fakeSessionVerifier struct {
	shared_ports.SessionVerifier
	session *session_domain.UserSession
	err     error
	calls   atomic.Int32
}

func (f *fakeSessionVerifier) Verify(_ context.Context, _ string) (*session_domain.UserSession, error) {
	f.calls.Add(1)
	return f.session, f.err
}

// newTestAPI arma un nexus.Server con un endpoint de negocio, uno de administración y uno interno,
// y los middlewares en el mismo orden que el composition root.
func newTestAPI(apiSecret string, verifier shared_ports.SessionVerifier) (http.Handler, *probe) {
	p := &probe{}
	server := &nexus.Server{
		ServerName: testServerName,
		Secret:     apiSecret,
		Settings:   &nexus.Settings{PathPrefix: "/api/v1"},
	}
	server.Group("/bonds", []nexus.Endpoint{
		{Path: "GET /", HandlerFunc: p.ServeHTTP},
	})
	server.Group("/admin", []nexus.Endpoint{
		{Path: "GET /bonds", HandlerFunc: middlewares_adapters.RequireBloomingAdmin(p.ServeHTTP)},
	})
	server.Group("/internal", []nexus.Endpoint{
		{Path: "POST /maturity", HandlerFunc: p.ServeHTTP, Options: nexus.EndpointOptions{NoRequiresAuthentication: true}},
	})

	server.Use(middlewares_adapters.RequestLog)
	server.Use(middlewares_adapters.ValidateApiSecret)
	server.Use(middlewares_adapters.VerifySession(verifier))

	// PrepareEndpoints llena el índice de rutas que leen EndpointIsPublic y NoRequiresAuthentication.
	mux := server.PrepareEndpoints()
	return server.ApplyMiddlewares(mux), p
}

// authHeaders son los headers de un request autenticado con el token dado.
func authHeaders(token string) map[string]string {
	return map[string]string{"x-secret": testAPISecret, "x-token": token}
}

// serve manda un request al handler y devuelve la respuesta.
func serve(h http.Handler, method, target string, headers map[string]string, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// assertError verifica el status y el código de error de una respuesta de nexus.
func assertError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status %d, se esperaba %d (body %s)", rec.Code, status, rec.Body)
	}
	var body nexus.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("el body no es un error de nexus: %v (body %s)", err, rec.Body)
	}
	if body.Message != code {
		t.Fatalf("código %q, se esperaba %q", body.Message, code)
	}
}

// newFakeOAuth levanta un ms-oauth falso y devuelve su OAUTH_URL.
func newFakeOAuth(t *testing.T, handler http.HandlerFunc) string {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv.URL + "/api/v1"
}

// captureLog redirige el log estándar a un buffer mientras dura el test.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(previous) })
	return &buf
}
