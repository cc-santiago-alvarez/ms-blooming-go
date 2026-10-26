package bondBound_application

import (
	"context"
	"strings"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	shared_application "github.com/futurapps/ms-blooming-go/internal/core/shared/application"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// ListByUserAndObject devuelve los bonos iniciados, madurados, bloqueados o canjeados del usuario en el
// producto, en el alcance de la sesión. Sin bondObjectID es ErrBondObjectIDRequired, y un id inválido,
// ErrInvalidID.
func (app *BondBoundApplication) ListByUserAndObject(ctx context.Context, userID, bondObjectID string) ([]*bondBound_domain.BondBound, error) {
	bondObjectID = strings.TrimSpace(bondObjectID)
	switch {
	case bondObjectID == "":
		return nil, bondBound_domain.ErrBondObjectIDRequired
	case !shared_domain.IsValidID(userID) || !shared_domain.IsValidID(bondObjectID):
		return nil, shared_domain.ErrInvalidID
	}
	return app.Repository.ListByUserAndObject(ctx, userID, bondObjectID, shared_application.AgencyScope(ctx))
}
