package bondObjects_handlers

import (
	"net/http"
	"strconv"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
)

// ChartByID responde las guías del producto del path que ve la sesión y, con ?simulation=true, su
// simulación diaria. Un simulation ausente o que no es booleano vale false.
func (handler *BondObjectsHandler) ChartByID(w http.ResponseWriter, r *http.Request) {
	id, ok := handlers_shared.PathID(w, r, "id")
	if !ok {
		return
	}
	simulation, _ := strconv.ParseBool(r.URL.Query().Get("simulation"))

	chart, err := handler.bondObjectApplication.ChartByID(r.Context(), id, simulation)
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, chart)
}
