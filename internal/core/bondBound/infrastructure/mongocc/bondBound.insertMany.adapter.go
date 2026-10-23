package bondBound_mongocc_repository

import (
	"context"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
)

// InsertMany guarda los bonos nuevos; una lista vacía no hace nada.
func (repo *BondBoundMongoccRepository) InsertMany(ctx context.Context, bounds []*bondBound_domain.BondBound) error {
	if len(bounds) == 0 {
		return nil
	}

	docs := make([]any, len(bounds))
	for i, b := range bounds {
		docs[i] = b
	}
	_, err := repo.DB.InsertMany(ctx, collectionName, docs)
	return err
}
