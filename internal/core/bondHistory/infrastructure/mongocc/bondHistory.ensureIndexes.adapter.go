package bondHistory_mongocc_repository

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// EnsureIndexes crea los índices del historial; es idempotente. Usa los nombres por defecto de Mongo
// para no chocar con un índice igual que ya exista en la colección compartida.
func (repo *BondHistoryMongoccRepository) EnsureIndexes(ctx context.Context) error {
	models := []mongo.IndexModel{
		// listado por producto o bono comprado
		{Keys: bson.D{
			{Key: "relatedId", Value: 1},
			{Key: "typeId", Value: 1},
			{Key: "agencyId", Value: 1},
			{Key: "isRemove", Value: 1},
			{Key: "createdAt", Value: -1},
		}},
		// auditoría del gateway por text
		{Keys: bson.D{{Key: "text", Value: 1}}},
	}

	_, err := repo.DB.GetCollection(collectionName).Indexes().CreateMany(ctx, models)
	return err
}
