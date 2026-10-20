package bondObject_mongocc_repository

import (
	"context"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// UpdateOne guarda los campos editables y updatedAt de o; agencyId y authorizedUserIds quedan siempre
// como arreglo. Devuelve false si el producto ya no existe o está borrado.
func (repo *BondObjectMongoccRepository) UpdateOne(ctx context.Context, o *bondObject_domain.BondObject) (bool, error) {
	authorized := o.AuthorizedUserIDs
	if authorized == nil {
		authorized = []string{}
	}

	filter := bson.M{"_id": o.ID, "isRemove": false}
	update := bson.M{"$set": bson.M{
		"name":                      o.Name,
		"description":               o.Description,
		"agencyId":                  o.AgencyID,
		"key":                       o.Key,
		"bondTypeId":                o.BondTypeID,
		"amount":                    o.Amount,
		"isPrivateBond":             o.IsPrivateBond,
		"authorizedUserIds":         authorized,
		"maturityTimeInDays":        o.MaturityTimeInDays,
		"maturityRewardsPercentage": o.MaturityRewardsPercentage,
		"active":                    o.Active,
		"available":                 o.Available,
		"saleInitDate":              o.SaleInitDate,
		"saleEndDate":               o.SaleEndDate,
		"earlyCashTerm":             o.EarlyCashTerm,
		"updatedAt":                 o.UpdatedAt,
	}}

	res, err := repo.DB.UpdateOne(ctx, collectionName, filter, update, nil)
	if err != nil {
		return false, err
	}
	return res.MatchedCount == 1, nil
}
