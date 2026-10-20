package bondObject_mongocc_repository

import (
	"context"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// FindOneByIDIncludingRemoved devuelve el producto aunque esté borrado, o nil, nil si no existe.
func (repo *BondObjectMongoccRepository) FindOneByIDIncludingRemoved(ctx context.Context, id string) (*bondObject_domain.BondObject, error) {
	return repo.findOne(ctx, bson.M{"_id": id})
}
