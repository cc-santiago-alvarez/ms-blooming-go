package bondUserSetting_application

import (
	"context"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
)

// GetByUser devuelve el setting vivo de userID, o nil, nil si no tiene; nunca lo crea.
func (app *BondUserSettingApplication) GetByUser(ctx context.Context, userID string) (*bondUserSetting_domain.BondUserSetting, error) {
	return app.Repository.FindByUser(ctx, userID)
}
