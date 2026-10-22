package bondUserSetting_mongocc_repository

import (
	"context"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// FindByUser devuelve el setting vivo de userID, o nil, nil si no tiene.
func (repo *BondUserSettingMongoccRepository) FindByUser(ctx context.Context, userID string) (*bondUserSetting_domain.BondUserSetting, error) {
	return repo.findOne(ctx, liveFilter(userID), options.FindOne().SetSort(LatestFirst()))
}
