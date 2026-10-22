package bondUserSettings_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
)

// List responde la página de settings vivos del alcance de la sesión, con page y limit del query.
func (handler *BondUserSettingsHandler) List(w http.ResponseWriter, r *http.Request) {
	page, err := handler.bondUserSettingApplication.List(r.Context(), handlers_shared.QueryPagination(r))
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, page)
}
