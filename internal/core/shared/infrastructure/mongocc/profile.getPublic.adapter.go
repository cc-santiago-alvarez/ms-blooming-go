package shared_mongocc_repository

import (
	"context"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// GetPublic devuelve los perfiles no borrados de ids con la proyección pública. Sin ids no consulta.
func (repo *ProfileMongoccRepository) GetPublic(ctx context.Context, ids ...string) ([]shared_domain.PublicProfile, error) {
	unique := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok || id == "" {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return []shared_domain.PublicProfile{}, nil
	}

	filter := bson.M{"_id": bson.M{"$in": unique}, "isRemove": false}
	opts := options.Find().SetProjection(publicProfileProjection)
	cursor, err := repo.DB.Find(ctx, profileCollection, filter, opts)
	if err != nil {
		return nil, err
	}

	profiles := []shared_domain.PublicProfile{}
	if err := cursor.All(ctx, &profiles); err != nil {
		return nil, err
	}
	return profiles, nil
}
