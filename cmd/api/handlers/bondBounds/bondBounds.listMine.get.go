package bondBounds_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
)

// ListMine responde la página de bonos del usuario de la sesión, con los statusId del query (uno o varios),
// page y limit.
func (handler *BondBoundsHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	var f bondBound_domain.MineFilters
	for _, s := range r.URL.Query()["statusId"] {
		f.StatusIDs = append(f.StatusIDs, bondBound_domain.BondBoundStatus(s))
	}

	page, err := handler.bondBoundApplication.ListMine(r.Context(), f, handlers_shared.QueryPagination(r))
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, page)
}
