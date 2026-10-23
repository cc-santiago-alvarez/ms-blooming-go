package bondBounds_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
)

// ChartGuides responde las guías de la gráfica del bono del path, si es del usuario de la sesión o la sesión
// es admin.
func (handler *BondBoundsHandler) ChartGuides(w http.ResponseWriter, r *http.Request) {
	id, ok := handlers_shared.PathID(w, r, "id")
	if !ok {
		return
	}

	chart, err := handler.bondBoundApplication.ChartGuides(r.Context(), id)
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, chart)
}
