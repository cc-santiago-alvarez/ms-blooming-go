package bondBounds_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
)

// BondBoundsHandler atiende las rutas de los bonos comprados.
type BondBoundsHandler struct {
	bondBoundApplication bondBound_ports.BondBoundApplicationPorts
}

// NewBondBoundsHandler registra las rutas de /bond-bounds. La compra y la compra masiva son del usuario de la
// sesión, que paga con una de sus cuentas.
func NewBondBoundsHandler(server *nexus.Server, bondBoundApplication bondBound_ports.BondBoundApplicationPorts) {
	handler := &BondBoundsHandler{bondBoundApplication: bondBoundApplication}

	server.Group("/bond-bounds", []nexus.Endpoint{
		{Path: "POST /purchase", HandlerFunc: handler.Purchase},
		{Path: "POST /bulk-purchase", HandlerFunc: handler.BulkPurchase},
	})
}

// respondError responde el error de bondBound con su status.
func respondError(w http.ResponseWriter, r *http.Request, err error) {
	handlers_shared.ResponseWithError(w, r, bondBound_domain.MapToHTTPStatus(err), err)
}
