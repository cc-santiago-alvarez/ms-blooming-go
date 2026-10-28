package bondBound_application

import (
	"context"
	"strings"

	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_application "github.com/futurapps/ms-blooming-go/internal/core/shared/application"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Timeline devuelve las compras de userID, o del usuario de la sesión si viene vacío, con las fechas en la
// zona de la sesión. Las propias son de todas sus agencias. Las de otro usuario exigen admin, si no es
// ErrForbidden, y solo son las del alcance de agencia de la sesión. Un id inválido es ErrInvalidID.
func (app *BondBoundApplication) Timeline(ctx context.Context, userID string) ([]bondBound_ports.TimelineItem, error) {
	session, err := sessionUser(ctx)
	if err != nil {
		return nil, err
	}

	var agencyID *string
	userID = strings.TrimSpace(userID)
	switch {
	case userID == "" || userID == session.UserID:
		userID = session.UserID
	case !shared_domain.IsValidID(userID):
		return nil, shared_domain.ErrInvalidID
	case !session.AdminBlooming:
		return nil, shared_domain.ErrForbidden
	default:
		agencyID = shared_application.AgencyScope(ctx)
	}
	return app.Repository.Timeline(ctx, userID, agencyID, session_domain.UtcDiff(ctx))
}
