package bondType_application

import (
	"context"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	shared_application "github.com/futurapps/ms-blooming-go/internal/core/shared/application"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// List devuelve los tipos no borrados del alcance de agencia de la sesión, por name. agencyIDs solo
// filtra si la agencia de la sesión es la principal; sin sesión no hay tipos. Un agencyID inválido es
// ErrInvalidID.
func (app *BondTypeApplication) List(ctx context.Context, agencyIDs []string) ([]*bondType_domain.BondType, error) {
	ids := shared_domain.AgencyIDs(agencyIDs).Normalize()
	for _, id := range ids {
		if !shared_domain.IsValidID(id) {
			return nil, shared_domain.ErrInvalidID
		}
	}

	if scope := shared_application.AgencyScope(ctx); scope != nil {
		if *scope == "" {
			return []*bondType_domain.BondType{}, nil
		}
		ids = []string{*scope}
	}
	return app.Repository.Find(ctx, ids, false)
}
