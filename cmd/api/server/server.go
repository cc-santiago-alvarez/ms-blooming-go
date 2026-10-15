package server

import (
	"github.com/codecraftkit/nexus"
)

// InitServer arma el servidor, inicializa los dominios, registra los handlers y arranca el cron.
func InitServer() *nexus.Server {
	// deps se usa al inicializar el primer dominio.
	server, _ := initServerConfig()

	// ====================== HEXAGONAL ARCHITECTURE =====================

	// Init domains. Un dominio va antes que los que usan su ApplicationPorts.
	// bondHistory → bondUserSetting → bondType → bondObject → bondBound

	// Init and start cron scheduler

	// Register handlers

	return server
}
