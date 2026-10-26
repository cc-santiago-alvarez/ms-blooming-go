package bondBounds_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
)

// Statuses responde los seis estados de un bono en orden, con su nombre.
func (handler *BondBoundsHandler) Statuses(w http.ResponseWriter, r *http.Request) {
	_ = nexus.ResponseWithJSON(w, http.StatusOK, handler.bondBoundApplication.Statuses())
}
