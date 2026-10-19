package server

import (
	bondHistory_application "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/application"
	bondHistory_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/infrastructure/mongocc"
)

// initBondHistoryDomain crea los índices del historial y arma su aplicación. Sus rutas viven en los
// handlers de bondObjects y bondBounds.
func initBondHistoryDomain(deps *serverDeps) {
	repository := &bondHistory_mongocc_repository.BondHistoryMongoccRepository{DB: deps.mongoDB}
	ensureIndexes("bondHistory", repository)

	deps.bondHistoryApplication = &bondHistory_application.BondHistoryApplication{
		Repository: repository,
	}
}
