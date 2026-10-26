package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/codecraftkit/nexus"
	"github.com/rs/cors"
)

// routesApp arma el servidor completo con API_DEBUG en debug.
func routesApp(t *testing.T, debug bool) http.Handler {
	t.Helper()
	server := newTestNexus(debug, cors.Options{})
	server.Group("/bonds", []nexus.Endpoint{
		{Path: "GET /", HandlerFunc: func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }},
	})
	return newTestApp(t, server).httpServer().Handler
}

// /_routes y /_routes/raw solo existen con API_DEBUG=true, y /_routes no pide x-secret. CA-00.43
func TestRoutesOnlyInDebug(t *testing.T) {
	t.Run("API_DEBUG=false", func(t *testing.T) {
		handler := routesApp(t, false)
		captureLog(t)
		for _, path := range []string{"/_routes", "/_routes/raw"} {
			if rec := serve(handler, http.MethodGet, path, nil); rec.Code != http.StatusNotFound {
				t.Errorf("GET %s respondió %d, se esperaba 404", path, rec.Code)
			}
		}
	})

	t.Run("API_DEBUG=true", func(t *testing.T) {
		handler := routesApp(t, true)
		captureLog(t)

		rec := serve(handler, http.MethodGet, "/_routes", nil)

		if rec.Code != http.StatusOK {
			t.Fatalf("GET /_routes respondió %d, se esperaba 200", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "GET /api/v1/bonds") {
			t.Fatalf("GET /_routes no lista GET /api/v1/bonds: %s", rec.Body)
		}
	})
}
