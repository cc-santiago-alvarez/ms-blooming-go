package bondBound_application

import (
	"context"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_application "github.com/futurapps/ms-blooming-go/internal/core/shared/application"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// CountByStatus devuelve la suma de los montos y la cantidad de los bonos no borrados del alcance de la sesión
// que pasan f, con el nombre del estado por el que se filtró, o "" si no se filtró; el filtro de fecha usa la
// zona de la sesión. Un admin cuenta los de f.UserID o, sin él, los de todo el alcance; un no admin solo los
// suyos, y pedir los de otro usuario es ErrForbidden. Un filtro inválido es el error de su Validation.
func (app *BondBoundApplication) CountByStatus(ctx context.Context, f bondBound_domain.KpiFilters) (bondBound_ports.StatusCount, error) {
	f.Normalize()
	if err := f.Validation(); err != nil {
		return bondBound_ports.StatusCount{}, err
	}
	session, err := sessionUser(ctx)
	if err != nil {
		return bondBound_ports.StatusCount{}, err
	}

	// Un no admin solo ve sus propios números.
	switch {
	case f.UserID != "" && !shared_application.IsOwnerOrAdmin(ctx, f.UserID):
		return bondBound_ports.StatusCount{}, shared_domain.ErrForbidden
	case f.UserID == "" && !session.AdminBlooming:
		f.UserID = session.UserID
	}

	count, err := app.Repository.CountByStatus(ctx, shared_application.AgencyScope(ctx), f, session_domain.UtcDiff(ctx))
	if err != nil {
		return bondBound_ports.StatusCount{}, err
	}
	count.Status = f.StatusID.Name()
	return count, nil
}
