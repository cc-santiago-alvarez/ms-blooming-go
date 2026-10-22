package bondUserSettings_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
)

// GetMine responde el setting del usuario de la sesión con su cuenta, su perfil y el resumen de su
// portafolio; si no tiene, lo crea.
func (handler *BondUserSettingsHandler) GetMine(w http.ResponseWriter, r *http.Request) {
	detail, err := handler.bondUserSettingApplication.GetMine(r.Context())
	if err != nil {
		respondError(w, r, err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, detail)
}
