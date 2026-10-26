package server

import (
	"github.com/codecraftkit/nexus"
	jobs_handlers "github.com/futurapps/ms-blooming-go/cmd/api/handlers/jobs"
)

// registerJobHandlers registra las rutas internas de /internal/jobs, que dispara ms-cronjob-go.
func registerJobHandlers(server *nexus.Server, deps *serverDeps) {
	jobs_handlers.NewJobsHandler(server, deps.bondBoundApplication)
}
