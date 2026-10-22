package bondUserSettings_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
)

// SetSelectedAccount cambia la cuenta destino del usuario de la sesión por la del body y responde el
// setting.
func (handler *BondUserSettingsHandler) SetSelectedAccount(w http.ResponseWriter, r *http.Request) {
	var params bondUserSetting_domain.SetSelectedAccountParams
	if !handlers_shared.DecodeBody(w, r, &params) {
		return
	}

	s, err := handler.bondUserSettingApplication.SetSelectedAccount(r.Context(), params)
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, s)
}
