package bondUserSetting_application

import (
	"context"
	"errors"

	bondUserSetting_ports "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/ports"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_ports "github.com/futurapps/ms-blooming-go/internal/core/shared/ports"
)

// errNoSession es un caso de uso de usuario llamado sin sesión; el middleware la garantiza, así que es un
// error interno.
var errNoSession = errors.New("bondUserSetting: el contexto no tiene sesión de usuario")

// BondUserSettingApplication implementa los casos de uso de bondUserSetting.
type BondUserSettingApplication struct {
	Repository   bondUserSetting_ports.BondUserSettingMongoccRepositoryPorts
	BondBounds   bondUserSetting_ports.BoundSummaryPort
	FirstAccount bondUserSetting_ports.FirstAccountPort
	Balance      shared_ports.BalancePort
	Profiles     shared_ports.ProfileReader
}

// sessionUser devuelve la sesión del context, o errNoSession si no hay o no tiene usuario.
func sessionUser(ctx context.Context) (*session_domain.UserSession, error) {
	session, ok := session_domain.GetUserSession(ctx)
	if !ok || session.UserID == "" {
		return nil, errNoSession
	}
	return session, nil
}
