package shared_mongocc_repository

import (
	"context"
	"errors"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// GetAccount devuelve la cuenta si existe y es de userID; si no, nil, nil.
func (repo *BalanceMongoccRepository) GetAccount(ctx context.Context, accountID, userID string) (*shared_domain.BalanceAccount, error) {
	filter := bson.M{"_id": accountID, "userId": userID, "isRemove": false}
	opts := options.FindOne().SetProjection(bson.M{
		"_id": 1, "userId": 1, "agencyId": 1, "statusId": 1, "available": 1, "number": 1,
	})

	var account shared_domain.BalanceAccount
	if err := repo.DB.FindOne(ctx, balanceAccountCollection, filter, opts).Decode(&account); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}
