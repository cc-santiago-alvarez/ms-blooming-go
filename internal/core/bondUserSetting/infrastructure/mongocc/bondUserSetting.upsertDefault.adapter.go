package bondUserSetting_mongocc_repository

import (
	"context"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// UpsertDefault devuelve el setting vivo de s.UserID sin modificarlo; si no tiene, guarda s de forma
// atómica.
func (repo *BondUserSettingMongoccRepository) UpsertDefault(ctx context.Context, s *bondUserSetting_domain.BondUserSetting) (*bondUserSetting_domain.BondUserSetting, error) {
	onInsert := insertFields(s)
	onInsert["updatedAt"] = s.UpdatedAt

	return repo.upsertLive(ctx, s.UserID, bson.M{"$setOnInsert": onInsert})
}
