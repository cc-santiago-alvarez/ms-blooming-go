package bondObject_application

import (
	"context"

	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Update valida p, reemplaza los campos editables del producto no borrado y lo devuelve actualizado, o
// ErrBondObjectNotFound. Un bondTypeId inexistente o borrado es ErrBondObjectInvalidBondType. Registra en
// el historial los campos que cambiaron; si falla, solo se loguea. Los bonos ya comprados no cambian.
func (app *BondObjectApplication) Update(ctx context.Context, id string, p bondObject_domain.UpdateParams) (*bondObject_domain.BondObject, error) {
	p.Normalize()
	if err := p.Validation(); err != nil {
		return nil, err
	}

	o, err := app.findNotRemoved(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := app.checkBondType(ctx, p.BondTypeID); err != nil {
		return nil, err
	}

	before := *o
	o.Apply(p)
	ok, err := app.Repository.UpdateOne(ctx, o)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, bondObject_domain.ErrBondObjectNotFound
	}

	changes, edited := bondObject_domain.BuildChanges(before, *o)
	app.addHistory(ctx, bondHistory_domain.AddParams{
		RelatedID:        o.ID,
		Text:             shared_domain.TextBondUpdated,
		Changes:          historyChanges(changes),
		EditedProperties: edited,
	})
	return o, nil
}

// historyChanges pasa los cambios del producto al formato del historial.
func historyChanges(changes map[string]bondObject_domain.Change) map[string]bondHistory_domain.ChangeEntry {
	out := make(map[string]bondHistory_domain.ChangeEntry, len(changes))
	for field, c := range changes {
		out[field] = bondHistory_domain.ChangeEntry(c)
	}
	return out
}
