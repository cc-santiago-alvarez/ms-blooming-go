package bondObject_application

import (
	"context"

	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	shared_application "github.com/futurapps/ms-blooming-go/internal/core/shared/application"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Remove borra de forma lógica el producto del alcance de agencia de la sesión y lo registra en el
// historial; si el registro falla, solo se loguea. Si no existe o ya estaba borrado devuelve
// ErrBondObjectNotFound, si fuera de la agencia principal también es de otras agencias, ErrForbidden, y un
// id inválido es ErrInvalidID. Los bonos ya comprados no cambian.
func (app *BondObjectApplication) Remove(ctx context.Context, id string) error {
	o, err := app.findInScope(ctx, id)
	if err != nil {
		return err
	}
	if !shared_application.OnlyInAgencyScope(ctx, o.AgencyID...) {
		return shared_domain.ErrForbidden
	}

	ok, err := app.Repository.SoftDelete(ctx, o.ID)
	if err != nil {
		return err
	}
	if !ok {
		return bondObject_domain.ErrBondObjectNotFound
	}

	app.addHistory(ctx, bondHistory_domain.AddParams{RelatedID: o.ID, Text: shared_domain.TextBondRemove})
	return nil
}
