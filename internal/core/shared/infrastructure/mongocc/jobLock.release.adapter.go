package shared_mongocc_repository

import (
	"context"
	"log"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Release vence el lock name ahora si lo tiene Owner; si ya lo tomó otra réplica, no lo toca. Se escribe
// aunque ctx ya esté cancelado; un fallo solo se loguea y el lock vence con su ttl.
func (repo *JobLockMongoccRepository) Release(ctx context.Context, name string) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), bestEffortTimeout)
	defer cancel()

	now := shared_domain.Now()
	filter := bson.M{"_id": name, "owner": repo.Owner}
	update := bson.M{"$set": bson.M{"lockedUntil": now, "updatedAt": now}}
	if _, err := repo.DB.UpdateOne(ctx, jobLockCollection, filter, update, nil); err != nil {
		log.Printf("[JobLockPort] no se pudo liberar el lock %s: %v", name, err)
	}
}
