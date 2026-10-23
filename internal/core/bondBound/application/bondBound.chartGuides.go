package bondBound_application

import (
	"context"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_application "github.com/futurapps/ms-blooming-go/internal/core/shared/application"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// ChartGuides devuelve las guías del bono a partir de su createdAt, en la zona de la sesión. Si no existe
// es ErrBondBoundNotFound, y si no es del usuario ni es admin, ErrForbidden. Un id inválido es ErrInvalidID.
func (app *BondBoundApplication) ChartGuides(ctx context.Context, id string) (bondBound_domain.BondBoundChart, error) {
	b, err := app.findOwnedOrAdmin(ctx, id)
	if err != nil {
		return bondBound_domain.BondBoundChart{}, err
	}
	guides := shared_domain.BuildBoundGuides(b.ProjectionTerms(), b.CreatedAt, session_domain.UtcDiff(ctx))
	return bondBound_domain.BondBoundChart{Guides: guides}, nil
}

// findOwnedOrAdmin devuelve el bono no borrado si es del usuario de la sesión o si la sesión es admin. Si
// no existe es ErrBondBoundNotFound, y si es ajeno, ErrForbidden. Un id inválido es ErrInvalidID.
func (app *BondBoundApplication) findOwnedOrAdmin(ctx context.Context, id string) (*bondBound_domain.BondBound, error) {
	if !shared_domain.IsValidID(id) {
		return nil, shared_domain.ErrInvalidID
	}
	b, err := app.Repository.FindOneByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, bondBound_domain.ErrBondBoundNotFound
	}
	if !shared_application.IsOwnerOrAdmin(ctx, b.UserID) {
		return nil, shared_domain.ErrForbidden
	}
	return b, nil
}
