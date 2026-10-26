package server

import (
	"github.com/codecraftkit/mongocc"
	api_config "github.com/futurapps/ms-blooming-go/cmd/api/config"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	bondHistory_ports "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/ports"
	bondObject_application "github.com/futurapps/ms-blooming-go/internal/core/bondObject/application"
	bondType_application "github.com/futurapps/ms-blooming-go/internal/core/bondType/application"
	bondUserSetting_ports "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/ports"
	utils_adapters "github.com/futurapps/ms-blooming-go/internal/core/shared/adapters/utils"
	shared_ports "github.com/futurapps/ms-blooming-go/internal/core/shared/ports"
)

// serverDeps contiene la configuración, las conexiones, los repositorios y las aplicaciones que se
// cablean al inicializar el servidor.
type serverDeps struct {
	// ====================== Configuración ======================
	env *api_config.ApiEnv

	// ====================== Database Connections ======================
	// El *mongo.Client sale de mongoDB.Database().Client(), para desconectarlo al apagar.
	mongoDB *mongocc.MongoQueries

	// ====================== Adapters ======================
	sessionVerifier  shared_ports.SessionVerifier
	profileReader    shared_ports.ProfileReader
	agencyReader     shared_ports.AgencyReader
	exporterPort     shared_ports.ExporterPort
	activityPort     shared_ports.ActivityPort
	noticePort       shared_ports.NoticePort
	notificationPort shared_ports.NotificationPort
	utils            utils_adapters.UtilsPorts
	// transactionRunner es el mismo que usa balancePort, para que sus escrituras se sumen a la transacción.
	transactionRunner shared_ports.TransactionRunnerPort
	balancePort       shared_ports.BalancePort

	// ====================== Applications ======================
	bondHistoryApplication     bondHistory_ports.BondHistoryApplicationPorts
	bondUserSettingApplication bondUserSetting_ports.BondUserSettingApplicationPorts
	// bondType y bondObject se guardan concretos para que linkBondObjectToBondType pueda conectarlos.
	bondTypeApplication   *bondType_application.BondTypeApplication
	bondObjectApplication *bondObject_application.BondObjectApplication
	bondBoundApplication  bondBound_ports.BondBoundApplicationPorts
}
