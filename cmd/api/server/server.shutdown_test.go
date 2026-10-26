package server

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/codecraftkit/nexus"
	"github.com/rs/cors"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// freePort devuelve un puerto TCP libre de 127.0.0.1.
func freePort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	return port
}

// waitHealthy espera a que url responda 200.
func waitHealthy(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if resp, err := http.Get(url); err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("%s no respondió 200 en 5 s", url)
}

type slowResult struct {
	status int
	body   string
	err    error
}

// Con un SIGTERM y un request de 2 s en curso, el request termina con 200, Run vuelve sin error (el
// proceso termina con 0) después del request y Mongo queda desconectado.
// CA-00.25
func TestRunGracefulShutdown(t *testing.T) {
	started := make(chan struct{})
	handlerDone := make(chan time.Time, 1)
	var once sync.Once

	server := newTestNexus(false, cors.Options{})
	server.Port = freePort(t)
	server.Group("/slow", []nexus.Endpoint{
		{Path: "GET /", HandlerFunc: func(w http.ResponseWriter, _ *http.Request) {
			once.Do(func() { close(started) })
			time.Sleep(2 * time.Second)
			_, _ = w.Write([]byte("listo"))
			handlerDone <- time.Now()
		}},
	})
	app := newTestApp(t, server)
	buf := captureLog(t)

	runDone := make(chan time.Time, 1)
	go func() {
		app.Run()
		runDone <- time.Now()
	}()
	base := "http://127.0.0.1:" + server.Port
	waitHealthy(t, base+"/_health")

	slow := make(chan slowResult, 1)
	go func() {
		req, _ := http.NewRequest(http.MethodGet, base+"/api/v1/slow", nil)
		req.Header.Set("x-secret", testSecret)
		req.Header.Set("x-token", testToken)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			slow <- slowResult{err: err}
			return
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		slow <- slowResult{status: resp.StatusCode, body: string(body), err: err}
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("el request lento no llegó al handler")
	}

	// Run restaura el comportamiento por defecto tras la primera señal: se manda una sola.
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("no se pudo mandar SIGTERM: %v", err)
	}

	var r slowResult
	select {
	case r = <-slow:
	case <-time.After(10 * time.Second):
		t.Fatal("el request lento no terminó")
	}
	if r.err != nil || r.status != http.StatusOK || r.body != "listo" {
		t.Fatalf("el request en curso terminó con (%d, %q, %v), se esperaba (200, listo, nil)", r.status, r.body, r.err)
	}

	var runEnded time.Time
	select {
	case runEnded = <-runDone:
	case <-time.After(10 * time.Second):
		t.Fatal("Run no volvió después del SIGTERM")
	}
	finishedAt := <-handlerDone
	if runEnded.Before(finishedAt) {
		t.Fatal("Run volvió antes de que terminara el request en curso")
	}

	client := app.deps.mongoDB.Database().Client()
	if err := client.Ping(context.Background(), nil); !errors.Is(err, mongo.ErrClientDisconnected) {
		t.Fatalf("Ping() después del apagado = %v, se esperaba ErrClientDisconnected", err)
	}
	if resp, err := http.Get(base + "/_health"); err == nil {
		resp.Body.Close()
		t.Fatal("el servidor sigue aceptando conexiones después del apagado")
	}

	out := buf.String()
	if !strings.Contains(out, "señal recibida, apagando") || !strings.Contains(out, "apagado completo") {
		t.Fatalf("el log no muestra el apagado ordenado:\n%s", out)
	}
}
