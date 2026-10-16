package main

import (
	"bytes"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// childMarker hace que el binario de test corra main() en lugar de los tests.
const childMarker = "BLM_HEALTHCHECK_CHILD"

func TestMain(m *testing.M) {
	if os.Getenv(childMarker) == "1" {
		main()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// runHealthcheck corre el healthcheck en un subproceso, con un entorno vacío y el directorio de
// trabajo en un directorio temporal con files. Devuelve el código de salida, stderr y lo que tardó.
func runHealthcheck(t *testing.T, files map[string]string) (int, string, time.Duration) {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe)
	cmd.Dir = dir
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), childMarker + "=1"}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	start := time.Now()
	err = cmd.Run()
	elapsed := time.Since(start)
	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("no se pudo correr el subproceso: %v", err)
		}
		code = exitErr.ExitCode()
	}
	return code, stderr.String(), elapsed
}

// portOf devuelve el puerto de addr.
func portOf(addr string) string {
	_, port, _ := net.SplitHostPort(addr)
	return port
}

// healthServer levanta un API falso cuyo /_health responde status y anota la línea de cada request.
func healthServer(t *testing.T, status int) (string, <-chan string) {
	t.Helper()
	requests := make(chan string, 10)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Method + " " + r.URL.Path
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return portOf(srv.Listener.Addr().String()), requests
}

// closedPort devuelve un puerto en el que no escucha nadie.
func closedPort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := portOf(ln.Addr().String())
	ln.Close()
	return port
}

// silentPort devuelve un puerto que acepta conexiones y nunca responde.
func silentPort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var conns []net.Conn
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conns = append(conns, conn)
		}
	}()
	t.Cleanup(func() {
		ln.Close()
		for _, c := range conns {
			c.Close()
		}
	})
	return portOf(ln.Addr().String())
}

// envFiles arma .env con ENV=testing y .env.testing con API_PORT=port.
func envFiles(port string) map[string]string {
	return map[string]string{".env": "ENV=testing\n", ".env.testing": "API_PORT=" + port + "\n"}
}

// El healthcheck termina en 0 si /_health responde 200, y en 1 si no hay nadie escuchando en
// API_PORT, si responde otro status, si no responde a tiempo o si faltan los archivos o la variable.
// CA-00.26
func TestHealthcheck(t *testing.T) {
	t.Run("responde 200", func(t *testing.T) {
		t.Parallel()
		port, requests := healthServer(t, http.StatusOK)

		code, stderr, _ := runHealthcheck(t, envFiles(port))

		if code != 0 {
			t.Fatalf("salió con %d, se esperaba 0: %s", code, stderr)
		}
		if got := <-requests; got != "GET /_health" {
			t.Fatalf("el API recibió %q, se esperaba GET /_health", got)
		}
	})

	t.Run(".env.testing pisa a .env", func(t *testing.T) {
		t.Parallel()
		port, _ := healthServer(t, http.StatusOK)
		files := envFiles(port)
		files[".env"] = "ENV=testing\nAPI_PORT=" + closedPort(t) + "\n"

		if code, stderr, _ := runHealthcheck(t, files); code != 0 {
			t.Fatalf("salió con %d, se esperaba 0: %s", code, stderr)
		}
	})

	t.Run("nada escuchando", func(t *testing.T) {
		t.Parallel()
		code, stderr, _ := runHealthcheck(t, envFiles(closedPort(t)))
		if code != 1 || !strings.Contains(stderr, "healthcheck falló") {
			t.Fatalf("salió con %d y stderr %q, se esperaba 1 y healthcheck falló", code, stderr)
		}
	})

	t.Run("responde 503", func(t *testing.T) {
		t.Parallel()
		port, _ := healthServer(t, http.StatusServiceUnavailable)
		if code, stderr, _ := runHealthcheck(t, envFiles(port)); code != 1 || !strings.Contains(stderr, "503") {
			t.Fatalf("salió con %d y stderr %q, se esperaba 1 con el status", code, stderr)
		}
	})

	t.Run("acepta y no responde", func(t *testing.T) {
		t.Parallel()
		code, stderr, elapsed := runHealthcheck(t, envFiles(silentPort(t)))
		if code != 1 {
			t.Fatalf("salió con %d, se esperaba 1: %s", code, stderr)
		}
		if elapsed > 4*time.Second {
			t.Fatalf("tardó %s, el timeout total es de 3 s", elapsed)
		}
	})

	fileCases := []struct {
		name  string
		files map[string]string
	}{
		{name: "sin .env", files: map[string]string{".env.testing": "API_PORT=25099\n"}},
		{name: ".env sin ENV", files: map[string]string{".env": "API_PORT=25099\n", ".env.testing": "API_PORT=25099\n"}},
		{name: "falta .env.testing", files: map[string]string{".env": "ENV=testing\nAPI_PORT=25099\n"}},
		{name: "API_PORT vacía", files: map[string]string{".env": "ENV=testing\n", ".env.testing": "API_PORT=\n"}},
	}
	for _, tt := range fileCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if code, stderr, _ := runHealthcheck(t, tt.files); code != 1 {
				t.Fatalf("salió con %d, se esperaba 1: %s", code, stderr)
			}
		})
	}
}
