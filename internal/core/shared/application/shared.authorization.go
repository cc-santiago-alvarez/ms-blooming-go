package shared_application

import (
	"context"

	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
)

// IsOwnerOrAdmin indica si la sesión es dueña del recurso o tiene adminBlooming.
// Sin sesión devuelve false.
func IsOwnerOrAdmin(ctx context.Context, ownerUserID string) bool {
	session, ok := session_domain.GetUserSession(ctx)
	if !ok {
		return false
	}
	return session.AdminBlooming || (ownerUserID != "" && session.UserID == ownerUserID)
}

// AgencyScope devuelve la agencia por la que se filtra, o nil si la agencia de la sesión es la
// principal. Sin sesión devuelve un filtro vacío, que no encuentra nada.
func AgencyScope(ctx context.Context) *string {
	session, ok := session_domain.GetUserSession(ctx)
	if !ok {
		empty := ""
		return &empty
	}
	if session.Agency.IsMain {
		return nil
	}
	agencyID := session.AgencyID
	return &agencyID
}
