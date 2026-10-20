package server

import (
	bondType_application "github.com/futurapps/ms-blooming-go/internal/core/bondType/application"
	bondType_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/bondType/infrastructure/mongocc"
)

// initBondTypeDomain crea los índices de los tipos y arma su aplicación. El contador de productos se
// conecta después, con linkBondObjectToBondType.
func initBondTypeDomain(deps *serverDeps) {
	repository := &bondType_mongocc_repository.BondTypeMongoccRepository{DB: deps.mongoDB}
	ensureIndexes("bondType", repository)

	deps.bondTypeApplication = &bondType_application.BondTypeApplication{
		Repository: repository,
	}
}

// linkBondObjectToBondType le da a bondType el contador de productos de bondObject. Va aparte de los
// init porque cada dominio necesita al otro: bondObject valida tipos y bondType cuenta productos.
func linkBondObjectToBondType(deps *serverDeps) {
	deps.bondTypeApplication.BondObjects = deps.bondObjectApplication
}
