package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

func main() {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:25251", 2*time.Second)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "dial failed: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	// Timeout total de lectura+escritura
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))

	// Request HTTP/1.0 mínimo — sin keep-alive, sin headers extra
	_, err = conn.Write([]byte("GET /_health HTTP/1.0\r\nHost: localhost\r\n\r\n"))
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "write failed: %v\n", err)
		os.Exit(1)
	}

	// Solo necesitamos la primera línea: "HTTP/1.x 200 OK"
	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "read failed: %v\n", err)
		os.Exit(1)
	}

	statusLine := string(buf[:n])
	if idx := strings.Index(statusLine, "\r\n"); idx > 0 {
		statusLine = statusLine[:idx]
	}

	if !strings.HasPrefix(statusLine, "HTTP/1.0 200") &&
		!strings.HasPrefix(statusLine, "HTTP/1.1 200") {
		_, _ = fmt.Fprintf(os.Stderr, "unhealthy: %s\n", statusLine)
		os.Exit(1)
	}
}
