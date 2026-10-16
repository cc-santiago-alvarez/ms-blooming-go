package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"github.com/rs/cors"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// handlerApp arma el servidor completo con un único endpoint GET /api/v1/bonds.
func handlerApp(t *testing.T, handler http.HandlerFunc) http.Handler {
	t.Helper()
	server := newTestNexus(false, cors.Options{})
	server.Group("/bonds", []nexus.Endpoint{{Path: "GET /", HandlerFunc: handler}})
	return newTestApp(t, server, nil).httpServer().Handler
}

// assertErrorBody verifica el status y el código de error de una respuesta de nexus.
func assertErrorBody(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
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

// Un error que el dominio no mapea, como uno de Mongo, responde 500 ERR_INTERNAL y su texto solo va
// al log. CA-00.10
func TestUnmappedErrorIsInternal(t *testing.T) {
	mongoErr := fmt.Errorf("buscar bondBound: %w", mongo.CommandError{
		Code:    11000,
		Name:    "DuplicateKey",
		Message: "E11000 duplicate key error collection: futurapps.bondBound",
	})

	tests := []struct {
		name       string
		status     int
		err        error
		wantStatus int
		wantCode   string
		wantLogged string
	}{
		{name: "error de Mongo", status: shared_domain.MapToHTTPStatus(mongoErr), err: mongoErr,
			wantStatus: http.StatusInternalServerError, wantCode: "ERR_INTERNAL", wantLogged: "E11000"},
		{name: "error mapeado", status: shared_domain.MapToHTTPStatus(shared_domain.ErrAccountNotFound), err: shared_domain.ErrAccountNotFound,
			wantStatus: http.StatusNotFound, wantCode: "ERR_ACCOUNT_NOT_FOUND"},
		{name: "status sin definir", status: 0, err: errors.New("detalle-interno"),
			wantStatus: http.StatusInternalServerError, wantCode: "ERR_INTERNAL", wantLogged: "detalle-interno"},
		{name: "error nil", status: http.StatusInternalServerError, err: nil,
			wantStatus: http.StatusInternalServerError, wantCode: "ERR_INTERNAL", wantLogged: "<nil>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := handlerApp(t, func(w http.ResponseWriter, r *http.Request) {
				handlers_shared.ResponseWithError(w, r, tt.status, tt.err)
			})
			buf := captureLog(t)

			rec := serve(handler, http.MethodGet, "/api/v1/bonds", authHeaders())

			assertErrorBody(t, rec, tt.wantStatus, tt.wantCode)
			for _, detail := range []string{"E11000", "detalle-interno"} {
				if strings.Contains(rec.Body.String(), detail) {
					t.Fatalf("la respuesta expone el detalle %q: %s", detail, rec.Body)
				}
			}
			out := buf.String()
			if tt.wantLogged == "" {
				if strings.Contains(out, "[handler]") {
					t.Fatalf("un error mapeado no debe loguearse como interno:\n%s", out)
				}
			} else if !strings.Contains(out, "[handler] error interno en GET /api/v1/bonds") || !strings.Contains(out, tt.wantLogged) {
				t.Fatalf("el log no tiene el error interno con %q:\n%s", tt.wantLogged, out)
			}
			for _, secret := range []string{testSecret, testToken} {
				if strings.Contains(out, secret) {
					t.Fatalf("el log contiene %q:\n%s", secret, out)
				}
			}
		})
	}
}

// Un panic en un handler responde 500 ERR_INTERNAL; el valor del panic y el stack solo van al log.
// CA-00.42
func TestPanicIsInternal(t *testing.T) {
	t.Run("panic en el handler", func(t *testing.T) {
		handler := handlerApp(t, func(http.ResponseWriter, *http.Request) { panic("falla-secreta") })
		buf := captureLog(t)

		rec := serve(handler, http.MethodGet, "/api/v1/bonds", authHeaders())

		assertErrorBody(t, rec, http.StatusInternalServerError, "ERR_INTERNAL")
		if strings.Contains(rec.Body.String(), "falla-secreta") {
			t.Fatalf("la respuesta expone el panic: %s", rec.Body)
		}
		out := buf.String()
		if !strings.Contains(out, "panic: falla-secreta") || !strings.Contains(out, "goroutine") {
			t.Fatalf("el log no tiene el panic con su stack:\n%s", out)
		}
		if !strings.Contains(out, "[blooming-test] GET /api/v1/bonds 500 ") {
			t.Fatalf("RequestLog no registró el 500:\n%s", out)
		}
	})

	t.Run("ErrAbortHandler se vuelve a lanzar", func(t *testing.T) {
		handler := handlerApp(t, func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) })
		captureLog(t)

		recovered := func() (rec any) {
			defer func() { rec = recover() }()
			serve(handler, http.MethodGet, "/api/v1/bonds", authHeaders())
			return nil
		}()

		if recovered != http.ErrAbortHandler {
			t.Fatalf("recover() = %v, se esperaba http.ErrAbortHandler", recovered)
		}
	})
}
