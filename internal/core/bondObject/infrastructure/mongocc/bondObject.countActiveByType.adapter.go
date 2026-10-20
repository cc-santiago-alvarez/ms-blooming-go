package bondObject_mongocc_repository

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// CountActiveByType cuenta, por tipo y en una sola agregación, los productos no borrados y activos que
// userID puede ver, de cualquier agencia. Sin tipos no consulta; un tipo sin productos no aparece.
func (repo *BondObjectMongoccRepository) CountActiveByType(ctx context.Context, bondTypeIDs []string, userID string) (map[string]int, error) {
	counts := map[string]int{}
	if len(bondTypeIDs) == 0 {
		return counts, nil
	}

	pipeline := bson.A{
		bson.M{"$match": bson.M{
			"isRemove":   false,
			"active":     true,
			"bondTypeId": bson.M{"$in": bondTypeIDs},
			"$or":        visibleTo(userID),
		}},
		bson.M{"$group": bson.M{"_id": "$bondTypeId", "count": bson.M{"$sum": 1}}},
	}

	cursor, err := repo.DB.Aggregate(ctx, collectionName, pipeline, nil)
	if err != nil {
		return nil, err
	}

	var rows []struct {
		BondTypeID string `bson:"_id"`
		Count      int    `bson:"count"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	for _, r := range rows {
		counts[r.BondTypeID] = r.Count
	}
	return counts, nil
}
