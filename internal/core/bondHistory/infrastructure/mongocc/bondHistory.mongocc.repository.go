package bondHistory_mongocc_repository

import (
	"github.com/codecraftkit/mongocc"
	bondHistory_ports "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/ports"
)

// Colección del historial; la comparte con Meteor y la lee el gateway.
const collectionName = "bondHistory"

// BondHistoryMongoccRepository implementa BondHistoryMongoccRepositoryPorts sobre bondHistory.
type BondHistoryMongoccRepository struct {
	DB *mongocc.MongoQueries
}

var _ bondHistory_ports.BondHistoryMongoccRepositoryPorts = (*BondHistoryMongoccRepository)(nil)
