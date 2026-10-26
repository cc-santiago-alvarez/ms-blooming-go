package shared_mongocc_repository

import (
	"context"
	"time"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Acquire toma el lock name hasta ahora más ttl, a nombre de Owner, si no existe o ya venció. Si otro lo
// tiene vigente, el filtro no lo encuentra, el upsert choca con su _id y devuelve false.
func (repo *JobLockMongoccRepository) Acquire(ctx context.Context, name string, ttl time.Duration) (bool, error) {
	now := shared_domain.Now()
	filter := bson.M{"_id": name, "lockedUntil": bson.M{"$lt": now}}
	update := bson.M{"$set": bson.M{"lockedUntil": now.Add(ttl), "owner": repo.Owner, "updatedAt": now}}

	_, err := repo.DB.UpdateOne(ctx, jobLockCollection, filter, update, options.UpdateOne().SetUpsert(true))
	if mongo.IsDuplicateKeyError(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
