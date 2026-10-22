package bondUserSetting_mongocc_repository

import (
	"context"
	"errors"

	"github.com/codecraftkit/mongocc"
	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	bondUserSetting_ports "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Colección de las preferencias del inversionista; la comparte con Meteor.
const collectionName = "bondUserSetting"

// BondUserSettingMongoccRepository implementa BondUserSettingMongoccRepositoryPorts sobre bondUserSetting.
type BondUserSettingMongoccRepository struct {
	DB *mongocc.MongoQueries
}

var _ bondUserSetting_ports.BondUserSettingMongoccRepositoryPorts = (*BondUserSettingMongoccRepository)(nil)

// LatestFirst ordena los settings de un usuario del último modificado al primero. Es el que se usa
// mientras haya duplicados y el que conserva la auditoría.
func LatestFirst() bson.D {
	return bson.D{
		{Key: "updatedAt", Value: -1},
		{Key: "createdAt", Value: -1},
		{Key: "_id", Value: 1},
	}
}

// collection devuelve bondUserSetting decodificando un userId ObjectId legacy como su hex.
func (repo *BondUserSettingMongoccRepository) collection() *mongo.Collection {
	opts := options.Collection().SetBSONOptions(&options.BSONOptions{ObjectIDAsHexString: true})
	return repo.DB.Database().Collection(collectionName, opts)
}

// liveFilter es el setting vivo de userID.
func liveFilter(userID string) bson.M {
	return bson.M{"userId": userID, "isRemove": false}
}

// insertFields son los campos que guarda el alta de s; userId e isRemove salen del filtro del upsert.
func insertFields(s *bondUserSetting_domain.BondUserSetting) bson.M {
	return bson.M{
		"_id":                 s.ID,
		"agencyId":            s.AgencyID,
		"language":            s.Language,
		"selectedAccountId":   s.SelectedAccountID,
		"selectedBondBoundId": s.SelectedBondBoundID,
		"createdAt":           s.CreatedAt,
	}
}

// upsertLive aplica update al setting vivo de userID, creándolo si no existe, y devuelve cómo quedó. Si
// otra alta concurrente gana el índice único, reintenta una vez y actualiza la que ganó: Mongo no lo
// reintenta solo porque isRemove está en el filtro pero no en la clave del índice.
func (repo *BondUserSettingMongoccRepository) upsertLive(ctx context.Context, userID string, update bson.M) (*bondUserSetting_domain.BondUserSetting, error) {
	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After).
		SetSort(LatestFirst())

	s, err := repo.findOneAndUpdate(ctx, liveFilter(userID), update, opts)
	if mongo.IsDuplicateKeyError(err) {
		s, err = repo.findOneAndUpdate(ctx, liveFilter(userID), update, opts)
	}
	return s, err
}

// findOneAndUpdate ejecuta el FindOneAndUpdate y decodifica el resultado.
func (repo *BondUserSettingMongoccRepository) findOneAndUpdate(ctx context.Context, filter, update bson.M, opts *options.FindOneAndUpdateOptionsBuilder) (*bondUserSetting_domain.BondUserSetting, error) {
	var s bondUserSetting_domain.BondUserSetting
	if err := repo.collection().FindOneAndUpdate(ctx, filter, update, opts).Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

// findOne devuelve el setting que cumple filter, o nil, nil si no hay ninguno.
func (repo *BondUserSettingMongoccRepository) findOne(ctx context.Context, filter bson.M, opts *options.FindOneOptionsBuilder) (*bondUserSetting_domain.BondUserSetting, error) {
	var s bondUserSetting_domain.BondUserSetting
	if err := repo.collection().FindOne(ctx, filter, opts).Decode(&s); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}
