package bondBounds_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
)

// List responde la página de bonos del alcance de la sesión, filtrada por statusId, userId, search y el filtro
// de fecha del query, con page y limit.
func (handler *BondBoundsHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := bondBound_domain.ListFilters{
		StatusID: bondBound_domain.BondBoundStatus(q.Get("statusId")),
		UserID:   q.Get("userId"),
		Search:   q.Get("search"),
		Date:     queryDateFilter(q),
	}

	page, err := handler.bondBoundApplication.List(r.Context(), f, handlers_shared.QueryPagination(r))
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, page)
}
