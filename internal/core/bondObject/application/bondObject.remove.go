package bondObject_application

import (
	"context"

	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Remove borra el producto de forma lógica y lo registra en el historial; si el registro falla, solo se
// loguea. Si no existe o ya estaba borrado devuelve ErrBondObjectNotFound, y un id inválido es
// ErrInvalidID. Los bonos ya comprados no cambian.
func (app *BondObjectApplication) Remove(ctx context.Context, id string) error {
	if !shared_domain.IsValidID(id) {
		return shared_domain.ErrInvalidID
	}

	ok, err := app.Repository.SoftDelete(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return bondObject_domain.ErrBondObjectNotFound
	}

	app.addHistory(ctx, bondHistory_domain.AddParams{RelatedID: id, Text: shared_domain.TextBondRemove})
	return nil
}
