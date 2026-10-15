package session_domain

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"sync/atomic"
)

const (
	minUtcDiff = -12
	maxUtcDiff = 14
)

// countryUtcDiff es la diferencia horaria por país del legado.
var countryUtcDiff = map[string]int{"CO": -5, "VE": -4, "MX": -7, "US": -6}

// defaultUtcDiff es DEFAULT_UTC_DIFF; el composition root lo fija al arrancar.
var defaultUtcDiff atomic.Int32

func init() {
	defaultUtcDiff.Store(-5)
}

// SetDefaultUtcDiff fija la diferencia horaria que se usa cuando la sesión no permite resolverla.
func SetDefaultUtcDiff(hours int) {
	defaultUtcDiff.Store(int32(hours))
}

// OptionalUtcDiff es una diferencia horaria en horas que puede no venir en la sesión.
// Acepta un entero o un string con un entero dentro de [-12, 14]; cualquier otro valor queda
// como no válido y nunca hace fallar el decode.
type OptionalUtcDiff struct {
	Hours int
	Valid bool
}

func (o *OptionalUtcDiff) UnmarshalJSON(b []byte) error {
	*o = OptionalUtcDiff{}

	raw := bytes.TrimSpace(b)
	if len(raw) > 0 && raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return nil
		}
		raw = []byte(strings.TrimSpace(s))
	}

	hours, err := strconv.ParseFloat(string(raw), 64)
	if err != nil || hours != math.Trunc(hours) || hours < minUtcDiff || hours > maxUtcDiff {
		return nil
	}

	*o = OptionalUtcDiff{Hours: int(hours), Valid: true}
	return nil
}

func (o OptionalUtcDiff) MarshalJSON() ([]byte, error) {
	if !o.Valid {
		return []byte("null"), nil
	}
	return []byte(strconv.Itoa(o.Hours)), nil
}

// ResolveUtcDiff resuelve la diferencia horaria de la sesión: utcDiff o gmt de la sesión, después
// el país y por último fallback.
func ResolveUtcDiff(s *UserSession, fallback int) int {
	if s == nil {
		return fallback
	}
	if s.UtcDiff.Valid {
		return s.UtcDiff.Hours
	}
	if s.Gmt.Valid {
		return s.Gmt.Hours
	}
	if hours, ok := countryUtcDiff[strings.ToUpper(strings.TrimSpace(s.CountryCode))]; ok {
		return hours
	}
	return fallback
}

// UtcDiff devuelve la diferencia horaria de la sesión del context, o DEFAULT_UTC_DIFF si no hay
// sesión.
func UtcDiff(ctx context.Context) int {
	s, _ := GetUserSession(ctx)
	return ResolveUtcDiff(s, int(defaultUtcDiff.Load()))
}
