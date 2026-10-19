package bondHistory_mongocc_repository

import (
	"context"

	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// FindByRelated devuelve los registros no borrados de relatedID con ese tipo, del más reciente al más
// antiguo. agencyID nil no filtra por agencia.
func (repo *BondHistoryMongoccRepository) FindByRelated(ctx context.Context, typeID bondHistory_domain.HistoryType, relatedID string, agencyID *string) ([]*bondHistory_domain.BondHistory, error) {
	filter := bson.M{"relatedId": relatedID, "typeId": typeID, "isRemove": false}
	if agencyID != nil {
		filter["agencyId"] = *agencyID
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})

	cursor, err := repo.DB.Find(ctx, collectionName, filter, opts)
	if err != nil {
		return nil, err
	}

	histories := []*bondHistory_domain.BondHistory{}
	if err := cursor.All(ctx, &histories); err != nil {
		return nil, err
	}
	return histories, nil
}
