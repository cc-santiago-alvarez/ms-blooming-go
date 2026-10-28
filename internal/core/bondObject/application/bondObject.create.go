package bondObject_application

import (
	"context"
	"errors"

	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_application "github.com/futurapps/ms-blooming-go/internal/core/shared/application"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Create normaliza y valida p, guarda el producto nuevo y lo devuelve. Fuera de la agencia principal, p solo
// puede tener la agencia de la sesión; si no, es ErrForbidden. Un bondTypeId inexistente, borrado o fuera
// del alcance de agencia es ErrBondObjectInvalidBondType. Registra el alta en el historial y en la
// actividad, con el usuario y la agencia de la sesión; si alguno falla, solo se loguea.
func (app *BondObjectApplication) Create(ctx context.Context, p bondObject_domain.CreateParams) (*bondObject_domain.BondObject, error) {
	p.Normalize()
	if err := p.Validation(); err != nil {
		return nil, err
	}
	if !shared_application.OnlyInAgencyScope(ctx, p.AgencyID...) {
		return nil, shared_domain.ErrForbidden
	}
	if err := app.checkBondType(ctx, p.BondTypeID); err != nil {
		return nil, err
	}

	o := bondObject_domain.NewBondObject(p)
	if err := app.Repository.InsertOne(ctx, o); err != nil {
		return nil, err
	}

	app.addHistory(ctx, bondHistory_domain.AddParams{RelatedID: o.ID, Text: shared_domain.TextBondAdded})

	var userID, agencyID string
	if session, ok := session_domain.GetUserSession(ctx); ok {
		userID, agencyID = session.UserID, session.AgencyID
	}
	app.Activity.Add(ctx, shared_domain.Activity{
		Description: shared_domain.TextBondCreated,
		RelatedID:   o.ID,
		AgencyID:    agencyID,
		UserID:      userID,
		Params: map[string]any{
			"name":        o.Name,
			"description": o.Description,
			"amount":      o.Amount,
			"bondTypeId":  o.BondTypeID,
		},
	})
	return o, nil
}

// checkBondType verifica que el tipo exista, no esté borrado y esté en el alcance de agencia de la sesión;
// si no, es ErrBondObjectInvalidBondType.
func (app *BondObjectApplication) checkBondType(ctx context.Context, bondTypeID string) error {
	_, err := app.BondTypes.GetByID(ctx, bondTypeID)
	if errors.Is(err, bondType_domain.ErrBondTypeNotFound) {
		return bondObject_domain.ErrBondObjectInvalidBondType
	}
	return err
}
