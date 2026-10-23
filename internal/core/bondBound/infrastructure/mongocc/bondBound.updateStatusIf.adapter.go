package bondBound_mongocc_repository

import (
	"context"

	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// UpdateStatusIf pasa el bono a c.To con las fechas de c que no son nil, y actualiza updatedAt, solo si no
// está borrado, sigue en c.From y no tiene soldDate ni tookDate. Devuelve false si no lo encontró así.
func (repo *BondBoundMongoccRepository) UpdateStatusIf(ctx context.Context, id string, c bondBound_ports.StatusChange) (bool, error) {
	filter := bson.M{
		"_id":      id,
		"isRemove": false,
		"statusId": c.From,
		"soldDate": nil,
		"tookDate": nil,
	}

	set := bson.M{"statusId": c.To, "updatedAt": shared_domain.Now()}
	if c.SoldDate != nil {
		set["soldDate"] = *c.SoldDate
	}
	if c.TookDate != nil {
		set["tookDate"] = *c.TookDate
	}
	if c.MaturedAt != nil {
		set["matured"] = true
		set["maturedAt"] = *c.MaturedAt
	}

	res, err := repo.DB.UpdateOne(ctx, collectionName, filter, bson.M{"$set": set}, nil)
	if err != nil {
		return false, err
	}
	return res.MatchedCount == 1, nil
}
