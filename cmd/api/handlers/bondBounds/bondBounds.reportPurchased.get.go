package bondBounds_handlers

import (
	"net/http"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
)

// ReportPurchased responde el Excel de los bonos comprados de la agencia de la sesión, filtrado por statusId,
// userId, search y el filtro de fecha del query.
func (handler *BondBoundsHandler) ReportPurchased(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := bondBound_domain.ReportFilters{
		StatusID: bondBound_domain.BondBoundStatus(q.Get("statusId")),
		UserID:   q.Get("userId"),
		Search:   q.Get("search"),
		Date:     queryDateFilter(q),
	}

	file, err := handler.bondBoundApplication.ReportPurchased(r.Context(), f)
	if err != nil {
		respondError(w, r, err)
		return
	}
	respondReport(w, file)
}
