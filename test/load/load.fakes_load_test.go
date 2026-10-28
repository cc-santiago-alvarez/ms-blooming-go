//go:build load

package load_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
)

// Servicios falsos que usa el servicio.
var (
	oauth *httptest.Server
	// sink es ms-notification y ms-exporter: acepta cualquier POST con 200 y descarta el body.
	sink *httptest.Server
)

// startFakes levanta ms-oauth y el receptor de avisos.
func startFakes() {
	oauth = httptest.NewServer(http.HandlerFunc(serveSession))
	sink = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
}

// closeFakes apaga los servicios falsos.
func closeFakes() {
	oauth.Close()
	sink.Close()
}

// serveSession es GET /session de ms-oauth: responde la sesión del x-token, o 401 si el token o el secreto
// no son válidos.
func serveSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet || r.URL.Path != "/api/v1/session" {
		http.NotFound(w, r)
		return
	}
	session, ok := sessions[r.Header.Get("x-token")]
	if !ok || r.Header.Get("x-secret") != oauthSecret {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(session)
}
