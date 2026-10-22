package bondUserSettings_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
)

// SetSelectedBondBound cambia el bono seleccionado del usuario de la sesión por el del body, o lo
// deselecciona con null, y responde el setting.
func (handler *BondUserSettingsHandler) SetSelectedBondBound(w http.ResponseWriter, r *http.Request) {
	var params bondUserSetting_domain.SetSelectedBondBoundParams
	if !handlers_shared.DecodeBody(w, r, &params) {
		return
	}

	s, err := handler.bondUserSettingApplication.SetSelectedBondBound(r.Context(), params)
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, s)
}
