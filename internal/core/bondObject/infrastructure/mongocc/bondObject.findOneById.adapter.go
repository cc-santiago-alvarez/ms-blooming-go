package bondObject_mongocc_repository

import (
	"context"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// FindOneByID devuelve el producto no borrado, o nil, nil si no existe o está borrado.
func (repo *BondObjectMongoccRepository) FindOneByID(ctx context.Context, id string) (*bondObject_domain.BondObject, error) {
	return repo.findOne(ctx, bson.M{"_id": id, "isRemove": false})
}
