package bondObjects_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
	bondHistory_ports "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/ports"
	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	bondObject_ports "github.com/futurapps/ms-blooming-go/internal/core/bondObject/ports"
	middlewares_adapters "github.com/futurapps/ms-blooming-go/internal/core/shared/middlewares"
)

// BondObjectsHandler atiende las rutas de los productos y la de su historial.
type BondObjectsHandler struct {
	bondObjectApplication  bondObject_ports.BondObjectApplicationPorts
	bondHistoryApplication bondHistory_ports.BondHistoryApplicationPorts
}

// NewBondObjectsHandler registra las rutas de /bond-objects, con el historial de un producto, y el listado
// de productos de un tipo en /bond-types. Exigen adminBlooming todas menos las del inversionista: más
// comprados, últimos, por tipo y la gráfica de un producto.
func NewBondObjectsHandler(server *nexus.Server, bondObjectApplication bondObject_ports.BondObjectApplicationPorts,
	bondHistoryApplication bondHistory_ports.BondHistoryApplicationPorts) {
	handler := &BondObjectsHandler{
		bondObjectApplication:  bondObjectApplication,
		bondHistoryApplication: bondHistoryApplication,
	}
	admin := middlewares_adapters.RequireBloomingAdmin

	server.Group("/bond-objects", []nexus.Endpoint{
		{Path: "GET /", HandlerFunc: admin(handler.List)},
		{Path: "GET /grouped-by-type", HandlerFunc: admin(handler.ListGroupedByType)},
		{Path: "GET /most-purchased", HandlerFunc: handler.MostPurchased},
		{Path: "GET /latest", HandlerFunc: handler.Latest},
		{Path: "GET /{id}", HandlerFunc: admin(handler.GetByID)},
		{Path: "GET /{id}/investors", HandlerFunc: admin(handler.ListInvestors)},
		{Path: "GET /{id}/chart", HandlerFunc: handler.ChartByID},
		{Path: "GET /{id}/history", HandlerFunc: admin(handler.History)},
		{Path: "POST /", HandlerFunc: admin(handler.Create)},
		{Path: "POST /chart/preview", HandlerFunc: admin(handler.ChartPreview)},
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
