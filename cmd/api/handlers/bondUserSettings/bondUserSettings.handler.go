package bondUserSettings_handlers

import (
	"net/http"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	bondUserSetting_ports "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/ports"
)

// BondUserSettingsHandler atiende las rutas de las preferencias del inversionista.
type BondUserSettingsHandler struct {
	bondUserSettingApplication bondUserSetting_ports.BondUserSettingApplicationPorts
}

// NewBondUserSettingsHandler registra las rutas de las preferencias. La interna la llama Meteor al crear
// una cuenta: no lleva sesión, pero exige x-secret.
func NewBondUserSettingsHandler(server *nexus.Server, bondUserSettingApplication bondUserSetting_ports.BondUserSettingApplicationPorts) {
	handler := &BondUserSettingsHandler{bondUserSettingApplication: bondUserSettingApplication}
	internal := nexus.EndpointOptions{NoRequiresAuthentication: true}

	server.Group("/internal/bond-user-settings", []nexus.Endpoint{
		{Path: "PUT /{userId}/selected-account", HandlerFunc: handler.InternalSelectedAccount, Options: internal},
	})
}

// respondError responde el error de bondUserSetting con su status.
func respondError(w http.ResponseWriter, r *http.Request, err error) {
	handlers_shared.ResponseWithError(w, r, bondUserSetting_domain.MapToHTTPStatus(err), err)
}
