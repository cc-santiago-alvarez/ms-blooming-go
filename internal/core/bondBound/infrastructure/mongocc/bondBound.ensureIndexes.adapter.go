package bondBound_mongocc_repository

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// EnsureIndexes crea los índices de los bonos; es idempotente. Usa los nombres por defecto de Mongo para
// no chocar con un índice igual que ya exista en la colección compartida. key no lleva índice único: el
// legado ya puede tener claves repetidas.
func (repo *BondBoundMongoccRepository) EnsureIndexes(ctx context.Context) error {
	models := []mongo.IndexModel{
		// bonos del usuario
		{Keys: bson.D{
			{Key: "userId", Value: 1},
			{Key: "isRemove", Value: 1},
			{Key: "statusId", Value: 1},
			{Key: "createdAt", Value: -1},
		}},
		// bonos de la agencia
		{Keys: bson.D{
			{Key: "agencyId", Value: 1},
			{Key: "isRemove", Value: 1},
			{Key: "statusId", Value: 1},
			{Key: "createdAt", Value: -1},
		}},
		// bonos de un producto
		{Keys: bson.D{
			{Key: "bondObjectId", Value: 1},
			{Key: "isRemove", Value: 1},
			{Key: "statusId", Value: 1},
		}},
		// bonos por madurar
		{Keys: bson.D{
			{Key: "statusId", Value: 1},
			{Key: "maturityDate", Value: 1},
		}},
		// un número de clave por tipo; los bonos legacy sin keySeq no cuentan
		{
			Keys: bson.D{
				{Key: "bondTypeId", Value: 1},
				{Key: "keySeq", Value: 1},
			},
			Options: options.Index().
				SetUnique(true).
				SetPartialFilterExpression(bson.M{"keySeq": bson.M{"$exists": true}}),
		},
	}

	_, err := repo.DB.GetCollection(collectionName).Indexes().CreateMany(ctx, models)
	return err
}
