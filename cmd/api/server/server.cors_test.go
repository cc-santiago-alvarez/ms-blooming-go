package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/codecraftkit/nexus"
	middlewares_adapters "github.com/futurapps/ms-blooming-go/internal/core/shared/middlewares"
)

// corsHandler arma el handler HTTP completo del servidor, con CORS por fuera, para corsOrigins.
func corsHandler(t *testing.T, corsOrigins ...string) http.Handler {
	t.Helper()
	corsOptions, err := middlewares_adapters.NewCorsOptions(corsOrigins)
	if err != nil {
		t.Fatalf("NewCorsOptions(%q) = %v", corsOrigins, err)
	}
	server := newTestNexus(false, corsOptions)
	server.Group("/bonds", []nexus.Endpoint{
		{Path: "GET /", HandlerFunc: func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }},
	})
	return newTestApp(t, server, nil).httpServer().Handler
}

// preflight manda el OPTIONS que haría un navegador antes de un GET con x-secret y x-token.
func preflight(h http.Handler, origin string) (int, http.Header) {
	rec := serve(h, http.MethodOptions, "/api/v1/bonds", map[string]string{
		"Origin":                         origin,
		"Access-Control-Request-Method":  http.MethodGet,
		"Access-Control-Request-Headers": "x-secret, x-token",
	})
	return rec.Code, rec.Header()
}

// Solo los orígenes de CORS_ORIGINS reciben Access-Control-Allow-Origin, y el preflight se responde
// antes de exigir x-secret. CA-00.9
func TestCORS(t *testing.T) {
	handler := corsHandler(t, "https://app.futurapps.com", "*.finocompany.com")

	allowed := []string{
		"https://x.finocompany.com",
		"http://x.finocompany.com",
		"https://a.b.finocompany.com",
		"https://X.FINOCOMPANY.COM",
		"https://app.futurapps.com",
	}
	for _, origin := range allowed {
		t.Run("permitido "+origin, func(t *testing.T) {
			code, header := preflight(handler, origin)
			if code != http.StatusNoContent {
				t.Fatalf("preflight respondió %d, se esperaba 204", code)
			}
			if got := header.Get("Access-Control-Allow-Origin"); got != origin {
				t.Fatalf("Access-Control-Allow-Origin = %q, se esperaba %q", got, origin)
			}
			if header.Get("Access-Control-Allow-Credentials") != "true" {
				t.Fatal("falta Access-Control-Allow-Credentials: true")
			}
			if methods := header.Get("Access-Control-Allow-Methods"); !strings.Contains(methods, http.MethodGet) {
				t.Fatalf("Access-Control-Allow-Methods = %q, no permite GET", methods)
			}
			headers := strings.ToLower(header.Get("Access-Control-Allow-Headers"))
			if !strings.Contains(headers, "x-secret") || !strings.Contains(headers, "x-token") {
				t.Fatalf("Access-Control-Allow-Headers = %q, no permite x-secret y x-token", headers)
			}

			rec := serve(handler, http.MethodGet, "/api/v1/bonds", map[string]string{
				"Origin": origin, "x-secret": testSecret, "x-token": testToken,
			})
			if rec.Code != http.StatusOK || rec.Header().Get("Access-Control-Allow-Origin") != origin {
				t.Fatalf("GET respondió %d con Access-Control-Allow-Origin %q", rec.Code, rec.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}

	denied := []string{
		"https://evil.com",
		"https://finocompany.com",
		"https://finocompany.com.evil.com",
		"https://xfinocompany.com",
		"http://app.futurapps.com",
		"https://app.futurapps.com:8443",
		"null",
	}
	for _, origin := range denied {
		t.Run("rechazado "+origin, func(t *testing.T) {
			code, header := preflight(handler, origin)
			if code == http.StatusUnauthorized {
				t.Fatal("el preflight llegó a ValidateApiSecret")
			}
			if got := header.Get("Access-Control-Allow-Origin"); got != "" {
				t.Fatalf("Access-Control-Allow-Origin = %q, se esperaba vacío", got)
			}
		})
	}

	t.Run("CORS_ORIGINS=*", func(t *testing.T) {
		anyOrigin := corsHandler(t, "*")
		if _, header := preflight(anyOrigin, "https://cualquiera.test"); header.Get("Access-Control-Allow-Origin") != "https://cualquiera.test" {
			t.Fatal("con * se rechazó un origen")
		}
		if _, header := preflight(anyOrigin, "null"); header.Get("Access-Control-Allow-Origin") != "" {
			t.Fatal("con * se aceptó el origen null")
		}
	})

	t.Run("entradas mal formadas", func(t *testing.T) {
		for _, entry := range []string{"*.com", "a.*.com", "ftp://x.com", "https://x.com/ruta"} {
			if _, err := middlewares_adapters.NewCorsOptions([]string{"https://app.futurapps.com", entry}); err == nil {
				t.Errorf("NewCorsOptions aceptó %q", entry)
			}
		}
	})
}
