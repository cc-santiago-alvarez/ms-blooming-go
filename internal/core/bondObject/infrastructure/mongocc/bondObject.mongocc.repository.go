package bondObject_mongocc_repository

import (
	"context"
	"errors"

	"github.com/codecraftkit/mongocc"
	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	bondObject_ports "github.com/futurapps/ms-blooming-go/internal/core/bondObject/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Colección de los productos; la comparte con Meteor.
const collectionName = "bondObject"

// bondTypeCollection es la colección de los tipos, para el agrupado.
const bondTypeCollection = "bondType"

// BondObjectMongoccRepository implementa BondObjectMongoccRepositoryPorts sobre bondObject.
type BondObjectMongoccRepository struct {
	DB *mongocc.MongoQueries
}

var _ bondObject_ports.BondObjectMongoccRepositoryPorts = (*BondObjectMongoccRepository)(nil)

// publicValues son los valores de isPrivateBond con los que un producto es público: los falsos de JS. null
// también encuentra el campo ausente.
var publicValues = bson.A{false, nil, "", 0}

// visibleTo es la condición $or de privados: los públicos y, si hay usuario, los privados donde está
// autorizado.
func visibleTo(userID string) bson.A {
	conditions := bson.A{bson.M{"isPrivateBond": bson.M{"$in": publicValues}}}
	if userID != "" {
		conditions = append(conditions, bson.M{"authorizedUserIds": userID})
	}
	return conditions
}

// onlyAgency es la condición de agencyId de los productos que son solo de agencyID: el string igual, o un
// arreglo que la contiene y no tiene otra.
func onlyAgency(agencyID string) bson.M {
	return bson.M{"$eq": agencyID, "$not": bson.M{"$elemMatch": bson.M{"$ne": agencyID}}}
}

// findOne devuelve el producto que cumple filter, o nil, nil si no hay ninguno.
func (repo *BondObjectMongoccRepository) findOne(ctx context.Context, filter bson.M) (*bondObject_domain.BondObject, error) {
	var o bondObject_domain.BondObject
	if err := repo.DB.FindOne(ctx, collectionName, filter, nil).Decode(&o); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &o, nil
}
