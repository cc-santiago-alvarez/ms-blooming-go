package server

import (
	bondObject_application "github.com/futurapps/ms-blooming-go/internal/core/bondObject/application"
	bondObject_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/bondObject/infrastructure/mongocc"
)

// initBondObjectDomain crea los índices de los productos y arma su aplicación.
func initBondObjectDomain(deps *serverDeps) {
	repository := &bondObject_mongocc_repository.BondObjectMongoccRepository{DB: deps.mongoDB}
	ensureIndexes("bondObject", repository)

	deps.bondObjectApplication = &bondObject_application.BondObjectApplication{
		Repository: repository,
		BondTypes:  deps.bondTypeApplication,
		History:    deps.bondHistoryApplication,
		Activity:   deps.activityPort,
		Profiles:   deps.profileReader,
	}
}
