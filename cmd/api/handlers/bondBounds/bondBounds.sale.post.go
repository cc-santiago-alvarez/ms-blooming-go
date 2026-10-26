package bondBounds_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
)

// Sale vende o canjea el bono del path, que debe ser del usuario de la sesión, y responde lo acreditado y el
// saldo de la cuenta destino. No lleva body.
func (handler *BondBoundsHandler) Sale(w http.ResponseWriter, r *http.Request) {
	id, ok := handlers_shared.PathID(w, r, "id")
	if !ok {
		return
	}

	result, err := handler.bondBoundApplication.Sale(r.Context(), id)
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, result)
}
