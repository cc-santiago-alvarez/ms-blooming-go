package bondType_mongocc_repository

import (
	"context"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// UpdateOne guarda los campos editables y updatedAt de t; agencyId queda siempre como arreglo. Devuelve
// false si el tipo ya no existe o está borrado.
func (repo *BondTypeMongoccRepository) UpdateOne(ctx context.Context, t *bondType_domain.BondType) (bool, error) {
	filter := bson.M{"_id": t.ID, "isRemove": false}
	update := bson.M{"$set": bson.M{
		"name":        t.Name,
		"description": t.Description,
		"active":      t.Active,
		"agencyId":    t.AgencyID,
		"updatedAt":   t.UpdatedAt,
	}}

	res, err := repo.DB.UpdateOne(ctx, collectionName, filter, update, nil)
	if err != nil {
		return false, err
	}
	return res.MatchedCount == 1, nil
}
