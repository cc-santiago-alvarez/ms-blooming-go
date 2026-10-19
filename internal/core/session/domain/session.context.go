package session_domain

import "context"

type userSessionKeyType struct{}

var userSessionKey userSessionKeyType

type systemAgencyKeyType struct{}

var systemAgencyKey systemAgencyKeyType

// SetUserSession guarda la sesión en el context.
func SetUserSession(ctx context.Context, s *UserSession) context.Context {
	return context.WithValue(ctx, userSessionKey, s)
}

// GetUserSession devuelve la sesión del context, si existe.
func GetUserSession(ctx context.Context) (*UserSession, bool) {
	s, ok := ctx.Value(userSessionKey).(*UserSession)
	return s, ok && s != nil
}

// WithSystemAgency marca el context como un proceso del sistema, sin usuario, que actúa sobre
// agencyID.
func WithSystemAgency(ctx context.Context, agencyID string) context.Context {
	return context.WithValue(ctx, systemAgencyKey, agencyID)
}

// SystemAgency devuelve la agencia de un proceso del sistema; sin marca o con agencia vacía da false.
func SystemAgency(ctx context.Context) (string, bool) {
	agencyID, ok := ctx.Value(systemAgencyKey).(string)
	return agencyID, ok && agencyID != ""
}
