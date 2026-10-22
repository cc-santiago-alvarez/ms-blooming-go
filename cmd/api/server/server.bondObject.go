package server

import (
	"github.com/codecraftkit/nexus"
	bondObjects_handlers "github.com/futurapps/ms-blooming-go/cmd/api/handlers/bondObjects"
	bondObject_application "github.com/futurapps/ms-blooming-go/internal/core/bondObject/application"
	bondObject_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/bondObject/infrastructure/mongocc"
)

// initBondObjectDomain crea los índices de los productos y arma su aplicación.
func initBondObjectDomain(deps *serverDeps) {
	repository := &bondObject_mongocc_repository.BondObjectMongoccRepository{DB: deps.mongoDB}
	ensureIndexes("bondObject", repository)

	deps.bondObjectApplication = &bondObject_application.BondObjectApplication{
		Repository:     repository,
		BondBoundStats: &bondObject_mongocc_repository.BondBoundStatsMongoccRepository{DB: deps.mongoDB},
		BondTypes:      deps.bondTypeApplication,
		History:        deps.bondHistoryApplication,
		Activity:       deps.activityPort,
		Profiles:       deps.profileReader,
	}
}

// registerBondObjectHandlers registra las rutas de /bond-objects y la de productos por tipo.
func registerBondObjectHandlers(server *nexus.Server, deps *serverDeps) {
	bondObjects_handlers.NewBondObjectsHandler(server, deps.bondObjectApplication)
}
