package bondUserSetting_mongocc_repository

import (
	"context"
	"errors"

	"github.com/codecraftkit/mongocc"
	bondUserSetting_ports "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Colección de las cuentas de saldo del módulo legado; solo se lee.
const balanceAccountCollection = "balanceAccount"

// FirstAccountMongoccRepository implementa FirstAccountPort con una consulta de solo lectura sobre
// balanceAccount.
type FirstAccountMongoccRepository struct {
	DB *mongocc.MongoQueries
}

var _ bondUserSetting_ports.FirstAccountPort = (*FirstAccountMongoccRepository)(nil)

// FirstAccountID devuelve el _id de la cuenta no borrada más antigua de userID en agencyID, o nil si no
// tiene.
func (repo *FirstAccountMongoccRepository) FirstAccountID(ctx context.Context, userID, agencyID string) (*string, error) {
	filter := bson.M{"userId": userID, "agencyId": agencyID, "isRemove": false}
	opts := options.FindOne().
		SetSort(bson.D{{Key: "createdAt", Value: 1}, {Key: "_id", Value: 1}}).
		SetProjection(bson.M{"_id": 1})

	var account struct {
		ID string `bson:"_id"`
	}
	if err := repo.DB.FindOne(ctx, balanceAccountCollection, filter, opts).Decode(&account); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &account.ID, nil
}
