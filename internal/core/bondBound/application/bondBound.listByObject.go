package bondBound_application

import (
	"context"

	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	shared_application "github.com/futurapps/ms-blooming-go/internal/core/shared/application"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// ListByObject devuelve los bonos no borrados del producto en el alcance de la sesión, con el perfil público
// de cada dueño, leídos en una sola consulta. Un id inválido es ErrInvalidID.
func (app *BondBoundApplication) ListByObject(ctx context.Context, bondObjectID string) ([]bondBound_ports.BondBoundWithProfile, error) {
	if !shared_domain.IsValidID(bondObjectID) {
		return nil, shared_domain.ErrInvalidID
	}

	bounds, err := app.Repository.ListByObject(ctx, bondObjectID, shared_application.AgencyScope(ctx))
	if err != nil {
		return nil, err
	}

	userIDs := make([]string, len(bounds))
	for i, b := range bounds {
		userIDs[i] = b.UserID
	}
	profiles, err := app.Profiles.GetPublic(ctx, userIDs...)
	if err != nil {
		return nil, err
	}
	profilesByID := make(map[string]shared_domain.PublicProfile, len(profiles))
	for _, p := range profiles {
		profilesByID[p.ID] = p
	}

	items := make([]bondBound_ports.BondBoundWithProfile, len(bounds))
	for i, b := range bounds {
		items[i] = bondBound_ports.BondBoundWithProfile{BondBound: *b}
		if p, ok := profilesByID[b.UserID]; ok {
			items[i].Profile = &p
		}
	}
	return items, nil
}
