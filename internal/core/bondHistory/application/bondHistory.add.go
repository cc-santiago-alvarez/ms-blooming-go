package bondHistory_application

import (
	"context"
	"fmt"

	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
)

// Add registra el evento a nombre de la sesión o, sin sesión, del sistema. Usa el ctx recibido, así que
// dentro de una transacción se suma a ella; el error se devuelve para que el llamador decida.
func (app *BondHistoryApplication) Add(ctx context.Context, p bondHistory_domain.AddParams) error {
	p.Normalize()
	if err := p.Validation(); err != nil {
		return err
	}

	userID, agencyID, err := author(ctx)
	if err != nil {
		return err
	}
	return app.Repository.InsertOne(ctx, bondHistory_domain.NewBondHistory(p, userID, agencyID))
}

// author devuelve el usuario y la agencia de la sesión o, sin sesión, la agencia del sistema con usuario
// vacío.
func author(ctx context.Context) (userID, agencyID string, err error) {
	if session, ok := session_domain.GetUserSession(ctx); ok {
		return session.UserID, session.AgencyID, nil
	}
	if agencyID, ok := session_domain.SystemAgency(ctx); ok {
		return "", agencyID, nil
	}
	return "", "", fmt.Errorf("%w: sin sesión ni modo sistema", bondHistory_domain.ErrBondHistoryInvalidParams)
}
