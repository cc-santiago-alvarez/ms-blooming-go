package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/textproto"
	"os"
	"strings"
	"time"

	api_config "github.com/futurapps/ms-blooming-go/cmd/api/config"
)

const (
	dialTimeout  = 2 * time.Second
	totalTimeout = 3 * time.Second
)

// main consulta GET /_health del API local y termina en 1 si no responde 200.
func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "healthcheck falló: %v\n", err)
		os.Exit(1)
	}
}

// run lee API_PORT de los mismos archivos de entorno que el API y consulta su /_health.
func run() error {
	if err := api_config.LoadEnvFiles(); err != nil {
		return err
	}
	port := strings.TrimSpace(os.Getenv("API_PORT"))
	if port == "" {
		return errors.New("API_PORT está vacía")
	}
	return probe(net.JoinHostPort("127.0.0.1", port))
}

// probe hace GET /_health en addr y devuelve error si el status no es 200.
func probe(addr string) error {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, dialTimeout)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := conn.SetDeadline(start.Add(totalTimeout)); err != nil {
		return err
	}

	// HTTP/1.0 mínimo: sin keep-alive ni headers extra.
	if _, err := conn.Write([]byte("GET /_health HTTP/1.0\r\nHost: 127.0.0.1\r\n\r\n")); err != nil {
		return err
	}

	// Solo importa la primera línea: "HTTP/1.1 200 OK".
	statusLine, err := textproto.NewReader(bufio.NewReader(conn)).ReadLine()
	if err != nil {
		return err
	}
	if parts := strings.SplitN(statusLine, " ", 3); len(parts) < 2 || parts[1] != "200" {
		return fmt.Errorf("respuesta inesperada: %q", statusLine)
	}
	return nil
}
