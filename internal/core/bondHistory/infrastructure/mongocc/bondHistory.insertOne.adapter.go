package bondHistory_mongocc_repository

import (
	"context"

	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
)

// InsertOne guarda el registro con el ctx recibido; dentro de una transacción se suma a ella.
func (repo *BondHistoryMongoccRepository) InsertOne(ctx context.Context, h *bondHistory_domain.BondHistory) error {
	_, err := repo.DB.InsertOne(ctx, collectionName, h)
	return err
}
