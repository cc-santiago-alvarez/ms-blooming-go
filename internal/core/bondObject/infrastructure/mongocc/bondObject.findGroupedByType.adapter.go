package bondObject_mongocc_repository

import (
	"context"

	bondObject_ports "github.com/futurapps/ms-blooming-go/internal/core/bondObject/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// FindGroupedByType agrupa por bondTypeId los productos no borrados, ordenados por name, y trae el tipo
// no borrado de cada grupo en la misma agregación. agencyIDs vacío no filtra por agencia.
func (repo *BondObjectMongoccRepository) FindGroupedByType(ctx context.Context, agencyIDs []string) ([]bondObject_ports.BondObjectGroup, error) {
	match := bson.M{"isRemove": false}
	if len(agencyIDs) > 0 {
		match["agencyId"] = bson.M{"$in": agencyIDs}
	}

	pipeline := bson.A{
		bson.M{"$match": match},
		bson.M{"$sort": bson.D{{Key: "name", Value: 1}}},
		bson.M{"$group": bson.M{"_id": "$bondTypeId", "list": bson.M{"$push": "$$ROOT"}}},
		bson.M{"$lookup": bson.M{
			"from": bondTypeCollection,
			"let":  bson.M{"id": "$_id"},
			"pipeline": bson.A{
				bson.M{"$match": bson.M{"$expr": bson.M{"$eq": bson.A{"$_id", "$$id"}}, "isRemove": false}},
			},
			"as": "bondType",
		}},
		// sin tipo vivo el campo no queda y se decodifica como nil
		bson.M{"$set": bson.M{"bondType": bson.M{"$arrayElemAt": bson.A{"$bondType", 0}}}},
		bson.M{"$sort": bson.D{{Key: "bondType.name", Value: 1}, {Key: "_id", Value: 1}}},
	}

	cursor, err := repo.DB.Aggregate(ctx, collectionName, pipeline, nil)
	if err != nil {
		return nil, err
	}

	groups := []bondObject_ports.BondObjectGroup{}
	if err := cursor.All(ctx, &groups); err != nil {
		return nil, err
	}
	return groups, nil
}
