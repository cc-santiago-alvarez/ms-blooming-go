package middlewares_adapters

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/rs/cors"
)

var allowedMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}

var allowedHeaders = []string{"Origin", "X-Requested-With", "Content-Type", "Accept", "X-Secret", "X-Token"}

// anyOrigin en CORS_ORIGINS acepta cualquier origen.
const anyOrigin = "*"

// AllowsAnyOrigin indica si CORS_ORIGINS abre a cualquier origen.
func AllowsAnyOrigin(origins []string) bool {
	return slices.Contains(origins, anyOrigin)
}

// NewCorsOptions arma la configuración de CORS a partir de CORS_ORIGINS. Devuelve error si alguna
// entrada está mal formada.
func NewCorsOptions(origins []string) (cors.Options, error) {
	allowAny := AllowsAnyOrigin(origins)

	var patterns []*regexp.Regexp
	for _, entry := range origins {
		if entry == anyOrigin {
			continue
		}
		re, err := compileOrigin(entry)
		if err != nil {
			return cors.Options{}, err
		}
		patterns = append(patterns, re)
	}

	allowOrigin := func(origin string) bool {
		origin = strings.TrimSuffix(strings.TrimSpace(origin), "/")
		if origin == "" || origin == "null" {
			return false
		}
		if allowAny {
			return true
		}
		for _, re := range patterns {
			if re.MatchString(origin) {
				return true
			}
		}
		return false
	}

	return cors.Options{
		AllowOriginFunc:  allowOrigin,
		AllowedMethods:   allowedMethods,
		AllowedHeaders:   allowedHeaders,
		AllowCredentials: true,
	}, nil
}

// compileOrigin convierte una entrada de CORS_ORIGINS en un regex anclado. Sin esquema acepta http
// y https; "*." al inicio del host acepta uno o más subdominios.
func compileOrigin(entry string) (*regexp.Regexp, error) {
	entry = strings.ToLower(entry)

	scheme := `https?://`
	host := entry
	if before, after, ok := strings.Cut(entry, "://"); ok {
		if before != "http" && before != "https" {
			return nil, fmt.Errorf("%q: el esquema debe ser http o https", entry)
		}
		scheme = regexp.QuoteMeta(before + "://")
		host = after
	}
	host = strings.TrimSuffix(host, "/")

	if host == "" || strings.ContainsAny(host, "/?# \t") {
		return nil, fmt.Errorf("%q: debe ser un origen sin ruta, por ejemplo https://app.dominio.com", entry)
	}

	hostPattern := regexp.QuoteMeta(host)
	if rest, ok := strings.CutPrefix(host, "*."); ok {
		if strings.Contains(rest, "*") || !strings.Contains(strings.Split(rest, ":")[0], ".") {
			return nil, fmt.Errorf("%q: el wildcard solo se admite como *.dominio.tld", entry)
		}
		hostPattern = `[a-z0-9-]+(\.[a-z0-9-]+)*\.` + regexp.QuoteMeta(rest)
	} else if strings.Contains(host, "*") {
		return nil, fmt.Errorf("%q: el wildcard solo se admite como *.dominio.tld", entry)
	}

	return regexp.Compile(`(?i)^` + scheme + hostPattern + `$`)
}
