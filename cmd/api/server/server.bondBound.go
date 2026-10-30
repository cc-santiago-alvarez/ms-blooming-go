package server

import (
	"log"

	"github.com/codecraftkit/nexus"
	bondBounds_handlers "github.com/futurapps/ms-blooming-go/cmd/api/handlers/bondBounds"
	bondBound_application "github.com/futurapps/ms-blooming-go/internal/core/bondBound/application"
	bondBound_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/bondBound/infrastructure/mongocc"
)

// initBondBoundDomain crea los índices de los bonos comprados y arma su aplicación. Usa el mismo
// transactionRunner que el saldo, para que el cobro y el depósito se sumen a la transacción de la compra y
// de la venta. Va después de bondUserSetting, que da la cuenta destino de las ventas. Avisa al arrancar si la
// compra y la venta están apagadas.
func initBondBoundDomain(deps *serverDeps) {
	repository := &bondBound_mongocc_repository.BondBoundMongoccRepository{DB: deps.mongoDB}
	ensureIndexes("bondBound", repository)

	deps.bondBoundApplication = &bondBound_application.BondBoundApplication{
		Repository:              repository,
		KeyCounter:              &bondBound_mongocc_repository.KeyCounterMongoccRepository{DB: deps.mongoDB},
		BondObjects:             deps.bondObjectApplication,
		BondTypes:               deps.bondTypeApplication,
		UserSettings:            deps.bondUserSettingApplication,
		Balance:                 deps.balancePort,
		History:                 deps.bondHistoryApplication,
		Tx:                      deps.transactionRunner,
		Utils:                   deps.utils,
		Notices:                 deps.noticePort,
		Notifications:           deps.notificationPort,
		Profiles:                deps.profileReader,
		Agencies:                deps.agencyReader,
		Exporter:                deps.exporterPort,
		JobLock:                 deps.jobLockPort,
		BulkPurchaseMaxQuantity: deps.env.BulkPurchaseMaxQuantity,
		AdminEmails:             deps.env.BloomingAdminEmails,
		TradingEnabled:          deps.env.BloomingTrading,
	}
	if !deps.env.BloomingTrading {
		log.Printf("[bondBound] compra y venta deshabilitadas: BLOOMING_TRADING no es true")
	}
}

// registerBondBoundHandlers registra las rutas de /bond-bounds.
func registerBondBoundHandlers(server *nexus.Server, deps *serverDeps) {
	bondBounds_handlers.NewBondBoundsHandler(server, deps.bondBoundApplication)
}
