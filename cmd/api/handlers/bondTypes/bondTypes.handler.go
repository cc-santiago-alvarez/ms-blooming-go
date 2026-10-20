package bondTypes_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	bondType_ports "github.com/futurapps/ms-blooming-go/internal/core/bondType/ports"
	middlewares_adapters "github.com/futurapps/ms-blooming-go/internal/core/shared/middlewares"
)

// BondTypesHandler atiende las rutas de los tipos de bono.
type BondTypesHandler struct {
	bondTypeApplication bondType_ports.BondTypeApplicationPorts
}

// NewBondTypesHandler registra las rutas de /bond-types; todas menos el listado de activos exigen
// adminBlooming.
func NewBondTypesHandler(server *nexus.Server, bondTypeApplication bondType_ports.BondTypeApplicationPorts) {
	handler := &BondTypesHandler{bondTypeApplication: bondTypeApplication}
	admin := middlewares_adapters.RequireBloomingAdmin

	server.Group("/bond-types", []nexus.Endpoint{
		{Path: "GET /", HandlerFunc: admin(handler.List)},
		{Path: "GET /active", HandlerFunc: handler.ListActive},
		{Path: "GET /{id}", HandlerFunc: admin(handler.GetByID)},
		{Path: "POST /", HandlerFunc: admin(handler.Create)},
		{Path: "PUT /{id}", HandlerFunc: admin(handler.Update)},
		{Path: "DELETE /{id}", HandlerFunc: admin(handler.Remove)},
	})
}

// respondError responde el error de bondType con su status.
func respondError(w http.ResponseWriter, r *http.Request, err error) {
	handlers_shared.ResponseWithError(w, r, bondType_domain.MapToHTTPStatus(err), err)
}
