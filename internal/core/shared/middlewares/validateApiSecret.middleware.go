package middlewares_adapters

import (
	"crypto/sha256"
	"crypto/subtle"
	"log"
	"net/http"
	"strings"
	"unicode"

	"github.com/codecraftkit/nexus"
)

// errInvalidSecret es la respuesta cuando x-secret falta o no coincide (RF-00.2).
const errInvalidSecret = "UNAUTHORIZED_INVALID_SECRET"

// ValidateApiSecret exige que el header x-secret coincida con API_SECRET (server.Secret) en todo
// endpoint que no sea IsPublic; los internos (NoRequiresAuthentication) también lo exigen (RF-00.2).
// La comparación es en tiempo constante sobre los SHA-256 de los dos valores normalizados, para no
// filtrar ni el contenido ni el largo del secret. Nunca se loguean headers ni valores.
func ValidateApiSecret(next http.Handler, server *nexus.Server) http.Handler {
	// nexus construye los middlewares una sola vez, al arrancar.
	normalizedSecret := normalizeSecret(server.Secret)
	expected := sha256.Sum256([]byte(normalizedSecret))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if server.EndpointIsPublic(r) {
			next.ServeHTTP(w, r)
			return
		}

		got := normalizeSecret(r.Header.Get("x-secret"))
		if got == "" || normalizedSecret == "" {
			rejectSecret(w, r, "ausente")
			return
		}

		digest := sha256.Sum256([]byte(got))
		if subtle.ConstantTimeCompare(digest[:], expected[:]) != 1 {
			rejectSecret(w, r, "inválido")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// rejectSecret responde 401 y deja en el log el motivo, el método y la ruta, sin valores.
func rejectSecret(w http.ResponseWriter, r *http.Request, reason string) {
	log.Printf("[ValidateApiSecret] x-secret %s en %s %s", reason, r.Method, r.URL.Path)
	_ = nexus.ResponseWithError(w, http.StatusUnauthorized, errInvalidSecret)
}

// normalizeSecret conserva solo letras, dígitos, puntuación y símbolos: quita espacios, tabs,
// saltos de línea y caracteres de control. Es el mismo criterio que petty-cash, para que un mismo
// x-secret valga igual en los dos servicios.
func normalizeSecret(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			return r
		}
		return -1
	}, s)
}
