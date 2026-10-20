package bondType_mongocc_repository

import (
	"github.com/codecraftkit/mongocc"
	bondType_ports "github.com/futurapps/ms-blooming-go/internal/core/bondType/ports"
)

// Colección de los tipos de bono; la comparte con Meteor.
const collectionName = "bondType"

// BondTypeMongoccRepository implementa BondTypeMongoccRepositoryPorts sobre bondType.
type BondTypeMongoccRepository struct {
	DB *mongocc.MongoQueries
}

var _ bondType_ports.BondTypeMongoccRepositoryPorts = (*BondTypeMongoccRepository)(nil)
