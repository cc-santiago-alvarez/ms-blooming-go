package server

import (
	"github.com/codecraftkit/nexus"
)

// InitServer carga la configuración, conecta Mongo y arma el nexus.Server con /api/v1
// (server.config.go); después inicializa los dominios, registra los handlers y arranca el cron
// (RF-00.1).
func InitServer() *nexus.Server {
	// deps se usa al inicializar el primer dominio.
	server, _ := initServerConfig()

	// ====================== HEXAGONAL ARCHITECTURE =====================

	// Init domains. Orden obligatorio (RF-00.1): un dominio se inicializa antes que los que usan
	// su ApplicationPorts.
	// bondHistory → bondUserSetting → bondType → bondObject → bondBound

	// Init and start cron scheduler (spec 10)

	// Register handlers

	return server
}
