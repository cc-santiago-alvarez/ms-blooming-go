package bondType_mongocc_repository

import (
	"context"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
)

// InsertOne guarda el tipo nuevo.
func (repo *BondTypeMongoccRepository) InsertOne(ctx context.Context, t *bondType_domain.BondType) error {
	_, err := repo.DB.InsertOne(ctx, collectionName, t)
	return err
}
