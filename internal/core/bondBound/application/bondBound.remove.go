package bondBound_application

import (
	"context"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	shared_application "github.com/futurapps/ms-blooming-go/internal/core/shared/application"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Remove borra el bono de forma lógica. Si no existe, ya estaba borrado o es de otra agencia que la de la
// sesión, que no es la principal, es ErrBondBoundNotFound. Un id inválido es ErrInvalidID.
func (app *BondBoundApplication) Remove(ctx context.Context, id string) error {
	if !shared_domain.IsValidID(id) {
		return shared_domain.ErrInvalidID
	}

	b, err := app.Repository.FindOneByID(ctx, id)
	if err != nil {
		return err
	}
	if b == nil || !shared_application.InAgencyScope(ctx, b.AgencyID) {
		return bondBound_domain.ErrBondBoundNotFound
	}

	ok, err := app.Repository.SoftDelete(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return bondBound_domain.ErrBondBoundNotFound
	}
	return nil
}
