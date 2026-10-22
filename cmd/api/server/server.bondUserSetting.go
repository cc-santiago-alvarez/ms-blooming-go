package server

import (
	"github.com/codecraftkit/nexus"
	bondUserSettings_handlers "github.com/futurapps/ms-blooming-go/cmd/api/handlers/bondUserSettings"
	bondUserSetting_application "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/application"
	bondUserSetting_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/infrastructure/mongocc"
)

// initBondUserSettingDomain crea los índices de las preferencias y arma su aplicación. Si hay usuarios con
// más de un setting vivo, el índice único no se crea y queda un warning: hay que correr
// cmd/tools/dedupusersettings.
func initBondUserSettingDomain(deps *serverDeps) {
	repository := &bondUserSetting_mongocc_repository.BondUserSettingMongoccRepository{DB: deps.mongoDB}
	ensureIndexes("bondUserSetting", repository)

	deps.bondUserSettingApplication = &bondUserSetting_application.BondUserSettingApplication{
		Repository:   repository,
		BondBounds:   &bondUserSetting_mongocc_repository.BoundSummaryMongoccRepository{DB: deps.mongoDB},
		FirstAccount: &bondUserSetting_mongocc_repository.FirstAccountMongoccRepository{DB: deps.mongoDB},
		Balance:      deps.balancePort,
		Profiles:     deps.profileReader,
	}
}

// registerBondUserSettingHandlers registra las rutas de /bond-user-settings y la interna que llama Meteor.
func registerBondUserSettingHandlers(server *nexus.Server, deps *serverDeps) {
	bondUserSettings_handlers.NewBondUserSettingsHandler(server, deps.bondUserSettingApplication)
}
