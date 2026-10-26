package bondBounds_handlers

import (
	"net/http"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
)

// ReportSummary responde el Excel del total comprado por cada inversionista de la agencia de la sesión,
// filtrado por el statusId del query; el resumen no usa los demás filtros.
func (handler *BondBoundsHandler) ReportSummary(w http.ResponseWriter, r *http.Request) {
	f := bondBound_domain.ReportFilters{
		StatusID: bondBound_domain.BondBoundStatus(r.URL.Query().Get("statusId")),
	}

	file, err := handler.bondBoundApplication.ReportSummary(r.Context(), f)
	if err != nil {
		respondError(w, r, err)
		return
	}
	respondReport(w, file)
}
