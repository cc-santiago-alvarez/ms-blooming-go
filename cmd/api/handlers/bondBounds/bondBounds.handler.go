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
)

// BondBoundsHandler atiende las rutas de los bonos comprados.
type BondBoundsHandler struct {
	bondBoundApplication bondBound_ports.BondBoundApplicationPorts
}

// NewBondBoundsHandler registra las rutas de /bond-bounds. La compra y la compra masiva son del usuario de la
// sesión, que paga con una de sus cuentas. Las guías y la cotización de un bono las ve su dueño o un admin, y
// solo el dueño lo vende; lo valida el caso de uso.
func NewBondBoundsHandler(server *nexus.Server, bondBoundApplication bondBound_ports.BondBoundApplicationPorts) {
	handler := &BondBoundsHandler{bondBoundApplication: bondBoundApplication}

	server.Group("/bond-bounds", []nexus.Endpoint{
		{Path: "GET /{id}/chart-guides", HandlerFunc: handler.ChartGuides},
		{Path: "GET /{id}/pre-sale", HandlerFunc: handler.PreSale},
		{Path: "POST /{id}/sale", HandlerFunc: handler.Sale},
		{Path: "POST /purchase", HandlerFunc: handler.Purchase},
		{Path: "POST /bulk-purchase", HandlerFunc: handler.BulkPurchase},
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
