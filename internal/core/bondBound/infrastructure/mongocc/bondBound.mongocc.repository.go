package bondBound_mongocc_repository

import (
	"github.com/codecraftkit/mongocc"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
)

// Colección de los bonos comprados; la comparte con Meteor.
const collectionName = "bondBound"

// BondBoundMongoccRepository implementa BondBoundMongoccRepositoryPorts sobre bondBound.
type BondBoundMongoccRepository struct {
	DB *mongocc.MongoQueries
}

var _ bondBound_ports.BondBoundMongoccRepositoryPorts = (*BondBoundMongoccRepository)(nil)
