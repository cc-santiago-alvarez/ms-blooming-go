package bondBound_mongocc_repository

import (
	"context"
	"errors"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// FindOneByID devuelve el bono no borrado, o nil, nil si no existe o está borrado.
func (repo *BondBoundMongoccRepository) FindOneByID(ctx context.Context, id string) (*bondBound_domain.BondBound, error) {
	var b bondBound_domain.BondBound
	filter := bson.M{"_id": id, "isRemove": false}
	if err := repo.DB.FindOne(ctx, collectionName, filter, nil).Decode(&b); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &b, nil
}
