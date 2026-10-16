package shared_exporter_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	shared_exporter "github.com/futurapps/ms-blooming-go/internal/core/shared/infrastructure/exporter"
)

const (
	testSecret   = "s3cr3t-exporter-valor"
	sessionToken = "tok-sesion-123"
	sensitive    = "dato-sensible-AB3"
	report       = "purchasedBonds"
	twoBytes     = `{"type":"Buffer","data":[80,75]}`
)

// received es lo que el ms-exporter falso recibió.
type received struct {
	method      string
	path        string
	contentType string
	secret      string
	token       []string
	body        string
}

// newFakeExporter levanta un ms-exporter que responde status con body y manda cada request al canal.
// Con hang, no responde hasta que el cliente corta. Devuelve su MS_EXPORTER_URL, con barra final.
func newFakeExporter(t *testing.T, status int, body string, hang bool) (string, <-chan received) {
	t.Helper()
	requests := make(chan received, 10)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqBody, _ := io.ReadAll(r.Body)
		requests <- received{
			method:      r.Method,
			path:        r.URL.EscapedPath(),
			contentType: r.Header.Get("Content-Type"),
			secret:      r.Header.Get("x-secret"),
			token:       r.Header.Values("x-token"),
			body:        string(reqBody),
		}
		if hang {
			select {
			case <-r.Context().Done():
			case <-time.After(10 * time.Second):
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/", requests
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

// sessionContext devuelve un context con la sesión del usuario que pide el reporte.
func sessionContext() context.Context {
	return session_domain.SetUserSession(context.Background(), &session_domain.UserSession{
		UserID: "u1",
		Token:  sessionToken,
	})
}

// reportPayload es el payload del reporte de bonos comprados, con un valor que nunca debe aparecer
// en el log.
func reportPayload() map[string]any {
	return map[string]any{
		"bondBounds": []map[string]any{{"key": "AB-3", "amount": 1500.5, "holder": sensitive}},
		"currency":   "$",
	}
}

// Con un 201 y buffer.toJSON(), Excel devuelve los bytes del archivo. CA-00.23
func TestExcel(t *testing.T) {
	file := make([]byte, 1000)
	for i := range file {
		file[i] = byte(i % 256)
	}
	copy(file, []byte{80, 75, 3, 4})
	numbers := make([]int, len(file))
	for i, b := range file {
		numbers[i] = int(b)
	}
	bigBody, err := json.Marshal(map[string]any{"type": "Buffer", "data": numbers})
	if err != nil {
		t.Fatalf("json.Marshal() = %v", err)
	}

	tests := []struct {
		name string
		body string
		want []byte
	}{
		{name: "dos bytes", body: twoBytes, want: []byte{80, 75}},
		{name: "archivo de 1000 bytes", body: string(bigBody), want: file},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseURL, _ := newFakeExporter(t, http.StatusCreated, tt.body, false)

			got, err := shared_exporter.NewExporterClient(baseURL, testSecret).Excel(sessionContext(), report, reportPayload())

			if err != nil {
				t.Fatalf("Excel() = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("Excel() devolvió %d bytes (%v…), se esperaban %d", len(got), got[:min(len(got), 8)], len(tt.want))
			}
		})
	}
}

// Cualquier falla de ms-exporter da ERR_EXPORT_FAILED sin detalle; el detalle solo va al log, sin
// el secret, el token ni el payload. CA-00.23
func TestExcelFailures(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		body         string
		down         bool
		hang         bool
		payload      any
		wantDetail   string
		wantRequests int
	}{
		{name: "responde 500", status: http.StatusInternalServerError, body: `{"message":"falla"}`, wantDetail: "respondió 500", wantRequests: 1},
		{name: "responde 200 con un Buffer", status: http.StatusOK, body: twoBytes, wantDetail: "respondió 200", wantRequests: 1},
		{name: "tipo distinto de Buffer", status: http.StatusCreated, body: `{"type":"Uint8Array","data":[80,75]}`, wantDetail: "no es un Buffer", wantRequests: 1},
		{name: "JSON inválido", status: http.StatusCreated, body: `{"type":"Buffer","data":[80,`, wantDetail: "respuesta inválida", wantRequests: 1},
		{name: "byte fuera de rango", status: http.StatusCreated, body: `{"type":"Buffer","data":[256]}`, wantDetail: "respuesta inválida", wantRequests: 1},
		{name: "ms-exporter caído", down: true},
		{name: "no responde antes del deadline", hang: true, wantDetail: "context deadline exceeded", wantRequests: 1},
		{name: "payload que no se serializa", status: http.StatusCreated, body: twoBytes, payload: func() {}, wantDetail: "unsupported type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseURL, requests := newFakeExporter(t, tt.status, tt.body, tt.hang)
			if tt.down {
				srv := httptest.NewServer(http.NotFoundHandler())
				srv.Close()
				baseURL = srv.URL
			}
			payload := tt.payload
			if payload == nil {
				payload = reportPayload()
			}
			ctx, cancel := context.WithTimeout(sessionContext(), 200*time.Millisecond)
			defer cancel()
			buf := captureLog(t)

			start := time.Now()
			got, err := shared_exporter.NewExporterClient(baseURL, testSecret).Excel(ctx, report, payload)
			elapsed := time.Since(start)

			if !errors.Is(err, shared_domain.ErrExportFailed) || err.Error() != "ERR_EXPORT_FAILED" || got != nil {
				t.Fatalf("Excel() = (%v, %v), se esperaba (nil, ERR_EXPORT_FAILED)", got, err)
			}
			if elapsed > 2*time.Second {
				t.Fatalf("Excel() tardó %s", elapsed)
			}
			if n := len(requests); n != tt.wantRequests {
				t.Fatalf("ms-exporter recibió %d requests, se esperaban %d", n, tt.wantRequests)
			}

			out := buf.String()
			if !strings.Contains(out, "[ExporterPort] no se pudo generar purchasedBonds: ") || !strings.Contains(out, tt.wantDetail) {
				t.Fatalf("log %q, se esperaba el reporte y %q", out, tt.wantDetail)
			}
			for _, secret := range []string{testSecret, sessionToken, sensitive} {
				if strings.Contains(out, secret) {
					t.Fatalf("el log contiene %q: %s", secret, out)
				}
			}
		})
	}
}

// Excel hace POST /export/excel/report/{report} con el payload sin envoltorio, x-secret y el x-token
// de la sesión. CA-00.41
func TestExcelRequest(t *testing.T) {
	const wantBody = `{"bondBounds":[{"key":"AB-3","amount":1500.5,"holder":"dato-sensible-AB3"}],"currency":"$"}`

	tests := []struct {
		name      string
		ctx       context.Context
		report    string
		wantPath  string
		wantToken []string
	}{
		{name: "con sesión", ctx: sessionContext(), report: report, wantPath: "/export/excel/report/purchasedBonds", wantToken: []string{sessionToken}},
		{name: "sin sesión", ctx: context.Background(), report: report, wantPath: "/export/excel/report/purchasedBonds"},
		{name: "reporte con barra", ctx: sessionContext(), report: "a/b", wantPath: "/export/excel/report/a%2Fb", wantToken: []string{sessionToken}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseURL, requests := newFakeExporter(t, http.StatusCreated, twoBytes, false)

			if _, err := shared_exporter.NewExporterClient(baseURL, testSecret).Excel(tt.ctx, tt.report, reportPayload()); err != nil {
				t.Fatalf("Excel() = %v", err)
			}

			r := <-requests
			if r.method != http.MethodPost || r.path != tt.wantPath {
				t.Fatalf("request %s %s, se esperaba POST %s", r.method, r.path, tt.wantPath)
			}
			if r.contentType != "application/json" || r.secret != testSecret {
				t.Fatalf("Content-Type %q y x-secret %q", r.contentType, r.secret)
			}
			if !reflect.DeepEqual(r.token, tt.wantToken) {
				t.Fatalf("x-token %q, se esperaba %q", r.token, tt.wantToken)
			}
			var got, want any
			if err := json.Unmarshal([]byte(r.body), &got); err != nil {
				t.Fatalf("body %q no es JSON: %v", r.body, err)
			}
			_ = json.Unmarshal([]byte(wantBody), &want)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("body %s, se esperaba %s", r.body, wantBody)
			}
		})
	}
}
