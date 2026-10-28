//go:build integration

package authz_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
)

// Servicios falsos que usa el servicio.
var (
	oauth         *httptest.Server
	notifications *httptest.Server
	exporter      *httptest.Server
)

// startFakes levanta ms-oauth, ms-notification y ms-exporter.
func startFakes() {
	oauth = httptest.NewServer(http.HandlerFunc(serveSession))
	notifications = httptest.NewServer(http.HandlerFunc(serveNotification))
	exporter = httptest.NewServer(http.HandlerFunc(serveExport))
}

// closeFakes apaga los servicios falsos.
func closeFakes() {
	oauth.Close()
	notifications.Close()
	exporter.Close()
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

// serveNotification es ms-notification: acepta los avisos con su secreto.
func serveNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.Header.Get("x-secret") != notificationSecret {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// serveExport es ms-exporter: con su secreto responde un Excel mínimo como el JSON del Buffer de Node, con
// 201, como el servicio real.
func serveExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.Header.Get("x-secret") != exporterSecret {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = io.WriteString(w, `{"type":"Buffer","data":[80,75,3,4]}`)
}
