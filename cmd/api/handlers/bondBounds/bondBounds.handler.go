package bondBounds_handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	middlewares_adapters "github.com/futurapps/ms-blooming-go/internal/core/shared/middlewares"
)

// BondBoundsHandler atiende las rutas de los bonos comprados.
type BondBoundsHandler struct {
	bondBoundApplication bondBound_ports.BondBoundApplicationPorts
}

// NewBondBoundsHandler registra las rutas de /bond-bounds, los bonos de un usuario en /users y los compradores
// de un producto en /bond-objects. Exigen adminBlooming el listado, la baja, los bonos de un usuario y los
// compradores. La compra, los estados, mis bonos, los recientes y la línea de tiempo son del usuario de la
// sesión. Un bono, su detalle, sus guías, su cotización y su historial los ve su dueño o un admin, y solo el
// dueño lo vende; lo valida el caso de uso.
func NewBondBoundsHandler(server *nexus.Server, bondBoundApplication bondBound_ports.BondBoundApplicationPorts) {
	handler := &BondBoundsHandler{bondBoundApplication: bondBoundApplication}
	admin := middlewares_adapters.RequireBloomingAdmin

	server.Group("/bond-bounds", []nexus.Endpoint{
		{Path: "GET /", HandlerFunc: admin(handler.List)},
		{Path: "GET /statuses", HandlerFunc: handler.Statuses},
		{Path: "GET /me", HandlerFunc: handler.ListMine},
		{Path: "GET /me/recent", HandlerFunc: handler.ListRecent},
		{Path: "GET /timeline", HandlerFunc: handler.Timeline},
		{Path: "GET /{id}", HandlerFunc: handler.GetByID},
		{Path: "GET /{id}/details", HandlerFunc: handler.GetDetails},
		{Path: "DELETE /{id}", HandlerFunc: admin(handler.Remove)},
		{Path: "GET /{id}/chart-guides", HandlerFunc: handler.ChartGuides},
		{Path: "GET /{id}/pre-sale", HandlerFunc: handler.PreSale},
		{Path: "POST /{id}/sale", HandlerFunc: handler.Sale},
		{Path: "GET /{id}/history", HandlerFunc: handler.History},
		{Path: "POST /purchase", HandlerFunc: handler.Purchase},
		{Path: "POST /bulk-purchase", HandlerFunc: handler.BulkPurchase},
	})
	server.Group("/users", []nexus.Endpoint{
		{Path: "GET /{userId}/bond-bounds", HandlerFunc: admin(handler.ListByUserAndObject)},
	})
	server.Group("/bond-objects", []nexus.Endpoint{
		{Path: "GET /{id}/bond-bounds", HandlerFunc: admin(handler.ListByObject)},
	})
}

// respondError responde el error de bondBound con su status. ERR_INVALID_WITHDRAW_DATE lleva en errors.detail
// el mensaje con la fecha de hoy.
func respondError(w http.ResponseWriter, r *http.Request, err error) {
	status := bondBound_domain.MapToHTTPStatus(err)
	if errors.Is(err, bondBound_domain.ErrInvalidWithdrawDate) {
		handlers_shared.ResponseWithErrorDetail(w, r, status, err, withdrawDateDetail(r.Context()))
		return
	}
	handlers_shared.ResponseWithError(w, r, status, err)
}

// withdrawDateDetail es el mensaje de una venta antes del día mínimo de retiro, con la fecha de hoy en la
// zona de la sesión.
func withdrawDateDetail(ctx context.Context) string {
	zone := time.FixedZone("", session_domain.UtcDiff(ctx)*3600)
	today := shared_domain.Now().In(zone).Format("02/01/2006")
	return shared_domain.Format(shared_domain.Text(shared_domain.TextBoundInvalidWithdrawDate), today)
}
