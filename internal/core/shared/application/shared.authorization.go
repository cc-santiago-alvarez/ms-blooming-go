package shared_application

import (
	"context"
	"slices"

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

// InAgencyScope indica si un recurso de agencyIDs está en el alcance de lectura de la sesión: la agencia
// principal ve todos y otra agencia, los que la incluyen. Sin sesión devuelve false.
func InAgencyScope(ctx context.Context, agencyIDs ...string) bool {
	scope := AgencyScope(ctx)
	if scope == nil {
		return true
	}
	return *scope != "" && slices.Contains(agencyIDs, *scope)
}

// OnlyInAgencyScope indica si un recurso de agencyIDs está en el alcance de escritura de la sesión: la
// agencia principal escribe todos y otra agencia, solo los que son únicamente suyos. Sin sesión devuelve
// false.
func OnlyInAgencyScope(ctx context.Context, agencyIDs ...string) bool {
	scope := AgencyScope(ctx)
	if scope == nil {
		return true
	}
	if *scope == "" || len(agencyIDs) == 0 {
		return false
	}
	for _, id := range agencyIDs {
		if id != *scope {
			return false
		}
	}
	return true
}
