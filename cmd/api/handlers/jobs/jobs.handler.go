package jobs_handlers

import (
	"github.com/codecraftkit/nexus"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
)

// JobsHandler atiende las rutas internas que disparan los jobs del ms.
type JobsHandler struct {
	bondBoundApplication bondBound_ports.BondBoundApplicationPorts
}

// NewJobsHandler registra las rutas de /internal/jobs. Las llama ms-cronjob-go: no llevan sesión, pero
// exigen x-secret.
func NewJobsHandler(server *nexus.Server, bondBoundApplication bondBound_ports.BondBoundApplicationPorts) {
	handler := &JobsHandler{bondBoundApplication: bondBoundApplication}
	internal := nexus.EndpointOptions{NoRequiresAuthentication: true}

	server.Group("/internal/jobs", []nexus.Endpoint{
		{Path: "POST /bond-maturity/run", HandlerFunc: handler.BondMaturity, Options: internal},
	})
}
