package bondBound_application

import (
	"context"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Chart devuelve las guías del bono a partir de su createdAt y su evolución desde initialDate hasta el
// vencimiento a la fecha de hoy, en la zona de la sesión. Si no existe o está fuera del alcance de agencia
// de la sesión es ErrBondBoundNotFound, y si no es del usuario ni es admin, ErrForbidden. Un id inválido es
// ErrInvalidID.
func (app *BondBoundApplication) Chart(ctx context.Context, id string) (bondBound_domain.BondBoundChart, error) {
	b, err := app.findOwnedOrAdmin(ctx, id)
	if err != nil {
		return bondBound_domain.BondBoundChart{}, err
	}
	terms, now, utcDiff := b.ProjectionTerms(), shared_domain.Now(), session_domain.UtcDiff(ctx)
	return bondBound_domain.BondBoundChart{
		Guides: shared_domain.BuildBoundGuides(terms, b.CreatedAt, utcDiff),
		Data:   shared_domain.BoundProjection(terms, b.Amount, b.InitialDate, now, utcDiff),
	}, nil
}
