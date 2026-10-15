package middlewares_adapters

import (
	"log"
	"net/http"
	"strings"

	"github.com/codecraftkit/nexus"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_ports "github.com/futurapps/ms-blooming-go/internal/core/shared/ports"
)

const (
	errTokenRequired = "TOKEN_IS_REQUIRED"
	errTokenInvalid  = "TOKEN_INVALID"
)

// VerifySession valida x-token contra ms-oauth y guarda la sesión en el context. Los endpoints
// NoRequiresAuthentication no la exigen. No loguea el token.
func VerifySession(verifier shared_ports.SessionVerifier) func(http.Handler, *nexus.Server) http.Handler {
	return func(next http.Handler, server *nexus.Server) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if server.NoRequiresAuthentication(r) {
				next.ServeHTTP(w, r)
				return
			}

			token := strings.TrimSpace(r.Header.Get("x-token"))
			if token == "" {
				log.Printf("[VerifySession] x-token ausente en %s %s", r.Method, r.URL.Path)
				_ = nexus.ResponseWithError(w, http.StatusUnauthorized, errTokenRequired)
				return
			}

			session, err := verifier.Verify(r.Context(), token)
			if err != nil {
				log.Printf("[VerifySession] sesión rechazada en %s %s: %v", r.Method, r.URL.Path, err)
				_ = nexus.ResponseWithError(w, http.StatusUnauthorized, errTokenInvalid)
				return
			}

			next.ServeHTTP(w, r.WithContext(session_domain.SetUserSession(r.Context(), session)))
		})
	}
}
