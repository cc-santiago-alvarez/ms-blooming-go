package middlewares_adapters

import (
	"log"
	"net/http"
	"time"

	"github.com/codecraftkit/nexus"
)

// statusWriter envuelve el ResponseWriter para quedarse con el status de la respuesta.
type statusWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

// WriteHeader guarda solo el primer status, que es el que net/http envía al cliente.
func (sw *statusWriter) WriteHeader(code int) {
	if !sw.wroteHeader {
		sw.statusCode = code
		sw.wroteHeader = true
	}
	sw.ResponseWriter.WriteHeader(code)
}

// Write sin WriteHeader previo envía un 200, que es el valor inicial de statusCode.
func (sw *statusWriter) Write(b []byte) (int, error) {
	sw.wroteHeader = true
	return sw.ResponseWriter.Write(b)
}

// Unwrap deja que http.ResponseController alcance el ResponseWriter real (flush y deadlines).
func (sw *statusWriter) Unwrap() http.ResponseWriter { return sw.ResponseWriter }

// Flush delega en el ResponseWriter real si implementa http.Flusher.
func (sw *statusWriter) Flush() {
	if flusher, ok := sw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// RequestLog registra método, ruta, status y duración de cada request, salvo /_health.
// No loguea headers, body ni query string.
func RequestLog(next http.Handler, server *nexus.Server) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_health" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(sw, r)

		log.Printf("[%s] %s %s %d %s",
			server.ServerName,
			r.Method,
			r.URL.Path,
			sw.statusCode,
			time.Since(start).Round(time.Microsecond),
		)
	})
}
