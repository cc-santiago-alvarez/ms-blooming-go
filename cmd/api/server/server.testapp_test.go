package server

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/codecraftkit/mongocc"
	"github.com/codecraftkit/nexus"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	middlewares_adapters "github.com/futurapps/ms-blooming-go/internal/core/shared/middlewares"
	shared_ports "github.com/futurapps/ms-blooming-go/internal/core/shared/ports"
	"github.com/rs/cors"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	testServerName = "blooming-test"
	testSecret     = "s3cr3t-server-valor"
	testToken      = "tok-server-valor"
)

// fakeVerifier acepta cualquier token con la misma sesión.
type fakeVerifier struct {
	shared_ports.SessionVerifier
}

func (fakeVerifier) Verify(context.Context, string) (*session_domain.UserSession, error) {
	return &session_domain.UserSession{UserID: "user-1", Token: testToken}, nil
}

// newTestNexus arma un nexus.Server como initServerConfig, con los middlewares en el mismo orden que
// en server.config.go.
func newTestNexus(debug bool, corsOptions cors.Options) *nexus.Server {
	server := &nexus.Server{
		ServerName:  testServerName,
		Secret:      testSecret,
		Debug:       debug,
		CorsOptions: corsOptions,
		Settings:    &nexus.Settings{PathPrefix: "/api/v1"},
	}
	server.Use(middlewares_adapters.RequestLog)
	server.Use(hideRoutesUnlessDebug)
	server.Use(recoverPanic)
	server.Use(middlewares_adapters.ValidateApiSecret)
	server.Use(middlewares_adapters.VerifySession(fakeVerifier{}))
	return server
}

// newTestApp arma el App con un cliente de Mongo que nunca llega a conectarse.
func newTestApp(t *testing.T, server *nexus.Server) *App {
	t.Helper()
	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://127.0.0.1:1"))
	if err != nil {
		t.Fatalf("no se pudo crear el cliente de Mongo: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })
	return &App{
		server: server,
		deps:   &serverDeps{mongoDB: mongocc.NewFromClient(client, "blooming_test", nil)},
	}
}

// authHeaders son los headers de un request autenticado.
func authHeaders() map[string]string {
	return map[string]string{"x-secret": testSecret, "x-token": testToken}
}

// serve manda un request al handler y devuelve la respuesta.
func serve(h http.Handler, method, target string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
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
