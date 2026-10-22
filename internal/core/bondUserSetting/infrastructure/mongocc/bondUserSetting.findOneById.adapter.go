package bondUserSetting_mongocc_repository

import (
	"context"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// FindOneByID devuelve el setting vivo, o nil, nil si no existe o está borrado.
func (repo *BondUserSettingMongoccRepository) FindOneByID(ctx context.Context, id string) (*bondUserSetting_domain.BondUserSetting, error) {
	return repo.findOne(ctx, bson.M{"_id": id, "isRemove": false}, nil)
}
