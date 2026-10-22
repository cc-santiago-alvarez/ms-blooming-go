package bondUserSettings_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
)

// InternalSelectedAccount cambia la cuenta destino del usuario del path por la del body, solo si ya tiene
// setting, y responde {updated}.
func (handler *BondUserSettingsHandler) InternalSelectedAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := handlers_shared.PathID(w, r, "userId")
	if !ok {
		return
	}
	var params bondUserSetting_domain.SetSelectedAccountParams
	if !handlers_shared.DecodeBody(w, r, &params) {
		return
	}

	updated, err := handler.bondUserSettingApplication.SyncSelectedAccount(r.Context(), userID, params)
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, map[string]bool{"updated": updated})
}
