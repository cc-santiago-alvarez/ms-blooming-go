package bondObjects_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	bondObject_ports "github.com/futurapps/ms-blooming-go/internal/core/bondObject/ports"
	middlewares_adapters "github.com/futurapps/ms-blooming-go/internal/core/shared/middlewares"
)

// BondObjectsHandler atiende las rutas de los productos.
type BondObjectsHandler struct {
	bondObjectApplication bondObject_ports.BondObjectApplicationPorts
}

// NewBondObjectsHandler registra las rutas de /bond-objects y el listado de productos de un tipo en
// /bond-types. Exigen adminBlooming todas menos las del inversionista: más comprados, últimos y por tipo.
func NewBondObjectsHandler(server *nexus.Server, bondObjectApplication bondObject_ports.BondObjectApplicationPorts) {
	handler := &BondObjectsHandler{bondObjectApplication: bondObjectApplication}
	admin := middlewares_adapters.RequireBloomingAdmin

	server.Group("/bond-objects", []nexus.Endpoint{
		{Path: "GET /", HandlerFunc: admin(handler.List)},
		{Path: "GET /grouped-by-type", HandlerFunc: admin(handler.ListGroupedByType)},
		{Path: "GET /most-purchased", HandlerFunc: handler.MostPurchased},
		{Path: "GET /latest", HandlerFunc: handler.Latest},
		{Path: "GET /{id}", HandlerFunc: admin(handler.GetByID)},
		{Path: "GET /{id}/investors", HandlerFunc: admin(handler.ListInvestors)},
		{Path: "POST /", HandlerFunc: admin(handler.Create)},
		{Path: "PUT /{id}", HandlerFunc: admin(handler.Update)},
		{Path: "PATCH /keys", HandlerFunc: admin(handler.UpdateKeys)},
		{Path: "DELETE /{id}", HandlerFunc: admin(handler.Remove)},
	})
	server.Group("/bond-types", []nexus.Endpoint{
		{Path: "GET /{bondTypeId}/bond-objects", HandlerFunc: handler.ListByType},
	})
}

// respondError responde el error de bondObject con su status.
func respondError(w http.ResponseWriter, r *http.Request, err error) {
	handlers_shared.ResponseWithError(w, r, bondObject_domain.MapToHTTPStatus(err), err)
}
