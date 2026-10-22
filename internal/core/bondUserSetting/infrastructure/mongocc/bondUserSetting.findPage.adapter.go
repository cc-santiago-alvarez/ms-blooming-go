package bondUserSetting_mongocc_repository

import (
	"context"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// FindPage devuelve la página p, ya normalizada, de los settings vivos por createdAt descendente.
// agencyID nil no filtra por agencia.
func (repo *BondUserSettingMongoccRepository) FindPage(ctx context.Context, agencyID *string, p shared_domain.Pagination) (shared_domain.Paginated[*bondUserSetting_domain.BondUserSetting], error) {
	var page shared_domain.Paginated[*bondUserSetting_domain.BondUserSetting]

	filter := bson.M{"isRemove": false}
	if agencyID != nil {
		filter["agencyId"] = *agencyID
	}

	total, err := repo.DB.CountDocuments(ctx, collectionName, filter)
	if err != nil {
		return page, err
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: 1}}).
		SetSkip(p.Skip()).
		SetLimit(int64(p.Limit))
	cursor, err := repo.collection().Find(ctx, filter, opts)
	if err != nil {
		return page, err
	}

	settings := []*bondUserSetting_domain.BondUserSetting{}
	if err := cursor.All(ctx, &settings); err != nil {
		return page, err
	}
	return shared_domain.NewPaginated(settings, total, p), nil
}
