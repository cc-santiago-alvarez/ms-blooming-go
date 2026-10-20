package bondObject_mongocc_repository

import (
	"context"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
)

// InsertOne guarda el producto nuevo.
func (repo *BondObjectMongoccRepository) InsertOne(ctx context.Context, o *bondObject_domain.BondObject) error {
	_, err := repo.DB.InsertOne(ctx, collectionName, o)
	return err
}
