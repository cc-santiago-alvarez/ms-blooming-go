package bondUserSetting_mongocc_repository

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// EnsureIndexes crea los índices de los settings; es idempotente. Usa los nombres por defecto de Mongo
// para no chocar con un índice igual que ya exista en la colección compartida. Si hay usuarios con más de
// un setting vivo, el índice único falla: antes hay que correr cmd/tools/dedupusersettings.
func (repo *BondUserSettingMongoccRepository) EnsureIndexes(ctx context.Context) error {
	models := []mongo.IndexModel{
		// un solo setting vivo por usuario
		{
			Keys: bson.D{{Key: "userId", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetPartialFilterExpression(bson.M{"isRemove": false}),
		},
		// listado de administración por agencia
		{Keys: bson.D{
			{Key: "agencyId", Value: 1},
			{Key: "isRemove", Value: 1},
			{Key: "createdAt", Value: -1},
		}},
	}

	_, err := repo.DB.GetCollection(collectionName).Indexes().CreateMany(ctx, models)
	return err
}
