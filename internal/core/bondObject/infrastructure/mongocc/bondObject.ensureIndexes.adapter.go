package bondObject_mongocc_repository

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// EnsureIndexes crea los índices de los productos; es idempotente. Usa los nombres por defecto de Mongo
// para no chocar con un índice igual que ya exista en la colección compartida.
func (repo *BondObjectMongoccRepository) EnsureIndexes(ctx context.Context) error {
	models := []mongo.IndexModel{
		// listados por agencia y tipo
		{Keys: bson.D{
			{Key: "isRemove", Value: 1},
			{Key: "agencyId", Value: 1},
			{Key: "bondTypeId", Value: 1},
			{Key: "active", Value: 1},
		}},
		// últimos publicados de la agencia
		{Keys: bson.D{
			{Key: "isRemove", Value: 1},
			{Key: "agencyId", Value: 1},
			{Key: "createdAt", Value: -1},
		}},
	}

	_, err := repo.DB.GetCollection(collectionName).Indexes().CreateMany(ctx, models)
	return err
}
