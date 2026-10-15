package session_domain

import "context"

type userSessionKeyType struct{}

var userSessionKey userSessionKeyType

// SetUserSession guarda la sesión en el context.
func SetUserSession(ctx context.Context, s *UserSession) context.Context {
	return context.WithValue(ctx, userSessionKey, s)
}

// GetUserSession devuelve la sesión del context, si existe.
func GetUserSession(ctx context.Context) (*UserSession, bool) {
	s, ok := ctx.Value(userSessionKey).(*UserSession)
	return s, ok && s != nil
}
