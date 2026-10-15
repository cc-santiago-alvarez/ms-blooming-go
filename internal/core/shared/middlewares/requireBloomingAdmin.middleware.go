package middlewares_adapters

import (
	"log"
	"net/http"

	"github.com/codecraftkit/nexus"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
)

const errForbidden = "ERR_FORBIDDEN"

// RequireBloomingAdmin deja pasar solo a sesiones con adminBlooming; al resto responde 403.
// Se aplica por endpoint al registrar las rutas.
func RequireBloomingAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, ok := session_domain.GetUserSession(r.Context())
		if !ok || !session.AdminBlooming {
			userID := ""
			if ok {
				userID = session.UserID
			}
			log.Printf("[RequireBloomingAdmin] acceso denegado a %q en %s %s", userID, r.Method, r.URL.Path)
			_ = nexus.ResponseWithError(w, http.StatusForbidden, errForbidden)
			return
		}
		next(w, r)
	}
}
