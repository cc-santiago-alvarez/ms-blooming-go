package shared_mongocc_repository

import (
	"context"
	"log"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Add guarda la actividad. Se escribe aunque ctx ya esté cancelado; un fallo solo se loguea.
func (repo *ActivityMongoccRepository) Add(ctx context.Context, a shared_domain.Activity) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), bestEffortTimeout)
	defer cancel()

	if _, err := repo.DB.InsertOne(ctx, activityCollection, shared_domain.NewActivityRecord(a)); err != nil {
		log.Printf("[ActivityPort] no se pudo guardar la actividad de %s: %v", a.RelatedID, err)
	}
}
