package bondBound_application

import (
	"context"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
)

// GetByID devuelve el bono no borrado si es del usuario de la sesión o la sesión es admin. Si no existe o
// está fuera del alcance de agencia de la sesión es ErrBondBoundNotFound, y si es ajeno, ErrForbidden. Un id
// inválido es ErrInvalidID.
func (app *BondBoundApplication) GetByID(ctx context.Context, id string) (*bondBound_domain.BondBound, error) {
	return app.findOwnedOrAdmin(ctx, id)
}
