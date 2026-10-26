package bondBound_mongocc_repository

import (
	"context"

	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// FindDetails devuelve el bono con su producto y su tipo, o nil, nil si no existe, está borrado o le falta el
// producto o el tipo. El producto sale sin los usuarios autorizados. No filtra por dueño ni por agencia.
func (repo *BondBoundMongoccRepository) FindDetails(ctx context.Context, id string) (*bondBound_ports.BondBoundDetailRow, error) {
	pipeline := bson.A{bson.M{"$match": bson.M{"_id": id, "isRemove": false}}, nestBound}
	pipeline = append(pipeline, lookupOne(bondObjectCollection, "bound.bondObjectId", "bondObject",
		bson.M{"authorizedUserIds": 0}, true)...)
	pipeline = append(pipeline, lookupOne(bondTypeCollection, "bound.bondTypeId", "bondType", nil, true)...)

	cursor, err := repo.DB.Aggregate(ctx, collectionName, pipeline, nil)
	if err != nil {
		return nil, err
	}

	var rows []bondBound_ports.BondBoundDetailRow
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}
