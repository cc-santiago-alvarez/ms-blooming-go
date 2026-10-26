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
// zona de la sesión. Las de otro usuario exigen admin; si no, ErrForbidden. Un id inválido es ErrInvalidID.
func (app *BondBoundApplication) Timeline(ctx context.Context, userID string) ([]bondBound_ports.TimelineItem, error) {
	session, err := sessionUser(ctx)
	if err != nil {
		return nil, err
	}

	userID = strings.TrimSpace(userID)
	switch {
	case userID == "":
		userID = session.UserID
	case !shared_domain.IsValidID(userID):
		return nil, shared_domain.ErrInvalidID
	case !shared_application.IsOwnerOrAdmin(ctx, userID):
		return nil, shared_domain.ErrForbidden
	}
	return app.Repository.Timeline(ctx, userID, session_domain.UtcDiff(ctx))
}
