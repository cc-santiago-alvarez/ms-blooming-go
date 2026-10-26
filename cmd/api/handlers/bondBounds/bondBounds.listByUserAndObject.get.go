package bondBounds_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
)

// ListByUserAndObject responde los bonos vigentes del usuario del path en el producto bondObjectId del query.
func (handler *BondBoundsHandler) ListByUserAndObject(w http.ResponseWriter, r *http.Request) {
	userID, ok := handlers_shared.PathID(w, r, "userId")
	if !ok {
		return
	}

	bounds, err := handler.bondBoundApplication.ListByUserAndObject(r.Context(), userID, r.URL.Query().Get("bondObjectId"))
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, bounds)
}
