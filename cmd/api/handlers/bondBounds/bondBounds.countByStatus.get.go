package bondBounds_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
)

// CountByStatus responde la suma de los montos y la cantidad de bonos, filtrados por statusId, userId, search y
// el filtro de fecha del query. Un no admin solo ve los suyos; lo valida el caso de uso.
func (handler *BondBoundsHandler) CountByStatus(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := bondBound_domain.KpiFilters{
		StatusID: bondBound_domain.BondBoundStatus(q.Get("statusId")),
		UserID:   q.Get("userId"),
		Search:   q.Get("search"),
		Date:     queryDateFilter(q),
	}

	count, err := handler.bondBoundApplication.CountByStatus(r.Context(), f)
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, count)
}
