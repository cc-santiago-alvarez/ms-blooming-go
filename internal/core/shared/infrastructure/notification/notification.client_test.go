package shared_notification_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_notification "github.com/futurapps/ms-blooming-go/internal/core/shared/infrastructure/notification"
)

const (
	testSecret   = "s3cr3t-notif-valor"
	sessionToken = "tok-sesion-123"
	sensitive    = "dato-sensible-AB3"
	group        = "user-ids"
	notification = "notify-blooming-bondPurchased"
)

// received es lo que el ms-notification falso recibió.
type received struct {
	method      string
	path        string
	contentType string
	secret      string
	token       []string
	body        string
}

// newFakeNotification levanta un ms-notification que responde status y manda cada request al canal.
// Devuelve su MS_NOTIFICATION_URL, con barra final.
func newFakeNotification(t *testing.T, status int) (string, <-chan received) {
	t.Helper()
	requests := make(chan received, 10)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requests <- received{
			method:      r.Method,
			path:        r.URL.EscapedPath(),
			contentType: r.Header.Get("Content-Type"),
			secret:      r.Header.Get("x-secret"),
			token:       r.Header.Values("x-token"),
			body:        string(body),
		}
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/api/v1/", requests
}

// downURL devuelve la URL de un servidor que ya no escucha.
func downURL() string {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()
	return srv.URL
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

// sessionContext devuelve un context con la sesión del usuario que dispara la notificación.
func sessionContext() context.Context {
	return session_domain.SetUserSession(context.Background(), &session_domain.UserSession{
		UserID: "u1",
		Token:  sessionToken,
	})
}

// purchaseParams son los params de una compra, con un valor que nunca debe aparecer en el log.
func purchaseParams() map[string]any {
	return map[string]any{"userIds": []string{"u1"}, "key": "AB-3", "quantity": 3, "amount": sensitive}
}

// assertJSONEqual compara dos JSON por su contenido.
func assertJSONEqual(t *testing.T, got, want string) {
	t.Helper()
	var g, w any
	if err := json.Unmarshal([]byte(got), &g); err != nil {
		t.Fatalf("body %q no es JSON: %v", got, err)
	}
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatalf("JSON esperado inválido %q: %v", want, err)
	}
	if !reflect.DeepEqual(g, w) {
		t.Fatalf("body %s, se esperaba %s", got, want)
	}
}

// Si ms-notification falla, Send no le devuelve nada al llamador y deja un warning en el log, sin
// el secret, el token ni los params. CA-00.19
func TestSendLogsFailures(t *testing.T) {
	failures := []struct {
		name       string
		status     int
		down       bool
		wantDetail string
	}{
		{name: "responde 500", status: http.StatusInternalServerError, wantDetail: "respondió 500"},
		{name: "responde 404", status: http.StatusNotFound, wantDetail: "respondió 404"},
		{name: "responde 401", status: http.StatusUnauthorized, wantDetail: "respondió 401"},
		{name: "ms-notification caído", down: true},
	}
	for _, tt := range failures {
		t.Run(tt.name, func(t *testing.T) {
			baseURL := downURL()
			if !tt.down {
				baseURL, _ = newFakeNotification(t, tt.status)
			}
			buf := captureLog(t)
			client := shared_notification.NewNotificationClient(baseURL, testSecret)

			start := time.Now()
			client.Send(sessionContext(), group, notification, purchaseParams())
			if elapsed := time.Since(start); elapsed > 2*time.Second {
				t.Fatalf("Send tardó %s", elapsed)
			}

			out := buf.String()
			lines := strings.Split(strings.TrimSpace(out), "\n")
			if len(lines) != 1 || out == "" {
				t.Fatalf("se esperaba una línea de log, hay %d:\n%s", len(lines), out)
			}
			wantPrefix := "[NotificationPort] no se pudo enviar user-ids/notify-blooming-bondPurchased: "
			if !strings.Contains(lines[0], wantPrefix) || !strings.Contains(lines[0], tt.wantDetail) {
				t.Fatalf("log %q, se esperaba %q con %q", lines[0], wantPrefix, tt.wantDetail)
			}
			for _, secret := range []string{testSecret, sessionToken, sensitive} {
				if strings.Contains(out, secret) {
					t.Fatalf("el log contiene %q: %s", secret, out)
				}
			}
		})
	}

	for _, status := range []int{http.StatusOK, http.StatusNoContent} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			baseURL, _ := newFakeNotification(t, status)
			buf := captureLog(t)

			shared_notification.NewNotificationClient(baseURL, testSecret).
				Send(sessionContext(), group, notification, purchaseParams())

			if buf.Len() != 0 {
				t.Fatalf("con %d no debía loguear nada: %s", status, buf)
			}
		})
	}
}

// Send manda {"data":{"params":…}} con x-secret aunque el ctx del llamador ya esté cancelado: con sesión
// a POST /notify/{group}/{notification} con su x-token, y sin sesión o sin token a
// POST /notify/internal/{group}/{notification} sin x-token. CA-00.40
func TestSendRequest(t *testing.T) {
	const fullBody = `{"data":{"params":{"userIds":["u1"],"key":"AB-3","quantity":3,"amount":"dato-sensible-AB3"}}}`
	const purchasePath = "/api/v1/notify/user-ids/notify-blooming-bondPurchased"
	const internalPath = "/api/v1/notify/internal/user-ids/notify-blooming-bondPurchased"

	cancelled := func(parent func() context.Context) func() context.Context {
		return func() context.Context {
			ctx, cancel := context.WithCancel(parent())
			cancel()
			return ctx
		}
	}

	tests := []struct {
		name         string
		ctx          func() context.Context
		group        string
		notification string
		params       map[string]any
		wantPath     string
		wantToken    []string
		wantBody     string
	}{
		{name: "con sesión", ctx: sessionContext, group: group, notification: notification,
			params: purchaseParams(), wantPath: purchasePath, wantToken: []string{sessionToken}, wantBody: fullBody},
		{name: "sin sesión", ctx: context.Background, group: group, notification: notification,
			params: purchaseParams(), wantPath: internalPath, wantBody: fullBody},
		{name: "sesión sin token", ctx: func() context.Context {
			return session_domain.SetUserSession(context.Background(), &session_domain.UserSession{UserID: "u1"})
		}, group: group, notification: notification, params: purchaseParams(), wantPath: internalPath, wantBody: fullBody},
		{name: "sin params", ctx: sessionContext, group: group, notification: notification,
			params: nil, wantPath: purchasePath, wantToken: []string{sessionToken}, wantBody: `{"data":{"params":{}}}`},
		{name: "grupo y notificación con barra", ctx: sessionContext, group: "a/b", notification: "c/d",
			params: purchaseParams(), wantPath: "/api/v1/notify/a%2Fb/c%2Fd", wantToken: []string{sessionToken}, wantBody: fullBody},
		{name: "sin sesión, grupo y notificación con barra", ctx: context.Background, group: "a/b", notification: "c/d",
			params: purchaseParams(), wantPath: "/api/v1/notify/internal/a%2Fb/c%2Fd", wantBody: fullBody},
		{name: "ctx ya cancelado", ctx: cancelled(sessionContext), group: group, notification: notification,
			params: purchaseParams(), wantPath: purchasePath, wantToken: []string{sessionToken}, wantBody: fullBody},
		{name: "sin sesión y ctx ya cancelado", ctx: cancelled(context.Background), group: group, notification: notification,
			params: purchaseParams(), wantPath: internalPath, wantBody: fullBody},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseURL, requests := newFakeNotification(t, http.StatusOK)
			buf := captureLog(t)

			shared_notification.NewNotificationClient(baseURL, testSecret).
				Send(tt.ctx(), tt.group, tt.notification, tt.params)

			var r received
			select {
			case r = <-requests:
			case <-time.After(5 * time.Second):
				t.Fatalf("ms-notification no recibió el request; log: %s", buf)
			}
			if r.method != http.MethodPost || r.path != tt.wantPath {
				t.Fatalf("request %s %s, se esperaba POST %s", r.method, r.path, tt.wantPath)
			}
			if r.contentType != "application/json" || r.secret != testSecret {
				t.Fatalf("Content-Type %q y x-secret %q", r.contentType, r.secret)
			}
			if !reflect.DeepEqual(r.token, tt.wantToken) {
				t.Fatalf("x-token %q, se esperaba %q", r.token, tt.wantToken)
			}
			assertJSONEqual(t, r.body, tt.wantBody)
			if buf.Len() != 0 {
				t.Fatalf("no debía loguear nada: %s", buf)
			}
		})
	}
}
