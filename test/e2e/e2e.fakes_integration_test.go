//go:build integration

package e2e_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// notifyTimeout es lo que se espera a un aviso: la compra y la venta avisan después de responder.
const notifyTimeout = 5 * time.Second

// excelFile es el Excel que devuelve ms-exporter, y excelBuffer, ese contenido como el JSON del Buffer de
// Node.
var (
	excelFile   = []byte{80, 75, 3, 4}
	excelBuffer = `{"type":"Buffer","data":[80,75,3,4]}`
)

// Servicios falsos que usa el servicio.
var (
	oauth         *httptest.Server
	notifications *fakeNotifications
	exporter      *fakeExporter
)

// startFakes levanta ms-oauth, ms-notification y ms-exporter.
func startFakes() {
	oauth = httptest.NewServer(http.HandlerFunc(serveSession))
	notifications = newFakeNotifications()
	exporter = newFakeExporter()
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

// fakeNotifications es ms-notification: acepta los avisos con su secreto y guarda el path de cada uno, sin
// el prefijo /api/v1.
type fakeNotifications struct {
	*httptest.Server
	mu    sync.Mutex
	paths []string
}

func newFakeNotifications() *fakeNotifications {
	f := &fakeNotifications{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("x-secret") != notificationSecret {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		f.mu.Lock()
		f.paths = append(f.paths, strings.TrimPrefix(r.URL.Path, "/api/v1"))
		f.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	return f
}

// received devuelve los paths de los avisos recibidos.
func (f *fakeNotifications) received() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.paths)
}

func (f *fakeNotifications) reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.paths = nil
}

// wait espera hasta notifyTimeout a que lleguen los avisos de paths, en cualquier orden.
func (f *fakeNotifications) wait(t *testing.T, paths ...string) {
	t.Helper()
	deadline := time.Now().Add(notifyTimeout)
	for {
		got := f.received()
		missing := slices.DeleteFunc(slices.Clone(paths), func(p string) bool { return slices.Contains(got, p) })
		if len(missing) == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("ms-notification recibió %v; faltan %v", got, missing)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// exportRequest es un request que recibió ms-exporter.
type exportRequest struct {
	path  string
	token string
	body  []byte
}

// fakeExporter es ms-exporter: con su secreto responde excelBuffer con 201, como el servicio real, y
// guarda cada request.
type fakeExporter struct {
	*httptest.Server
	mu       sync.Mutex
	requests []exportRequest
}

func newFakeExporter() *fakeExporter {
	f := &fakeExporter{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("x-secret") != exporterSecret {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		body, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.requests = append(f.requests, exportRequest{path: r.URL.Path, token: r.Header.Get("x-token"), body: body})
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, excelBuffer)
	}))
	return f
}

// received devuelve los requests recibidos.
func (f *fakeExporter) received() []exportRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.requests)
}

func (f *fakeExporter) reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = nil
}
