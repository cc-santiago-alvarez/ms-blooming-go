package server

import (
	"github.com/codecraftkit/mongocc"
	api_config "github.com/futurapps/ms-blooming-go/cmd/api/config"
	bondHistory_ports "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/ports"
	bondObject_application "github.com/futurapps/ms-blooming-go/internal/core/bondObject/application"
	bondType_application "github.com/futurapps/ms-blooming-go/internal/core/bondType/application"
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
	sessionVerifier shared_ports.SessionVerifier
	profileReader   shared_ports.ProfileReader

	// ====================== Applications ======================
	bondHistoryApplication bondHistory_ports.BondHistoryApplicationPorts
	// bondType y bondObject se guardan concretos para que linkBondObjectToBondType pueda conectarlos.
	bondTypeApplication   *bondType_application.BondTypeApplication
	bondObjectApplication *bondObject_application.BondObjectApplication
}
