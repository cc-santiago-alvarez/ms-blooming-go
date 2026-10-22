package bondUserSetting_application

import (
	"context"
	"errors"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
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

var _ bondUserSetting_ports.BondUserSettingApplicationPorts = (*BondUserSettingApplication)(nil)

// sessionUser devuelve la sesión del context, o errNoSession si no hay o no tiene usuario.
func sessionUser(ctx context.Context) (*session_domain.UserSession, error) {
	session, ok := session_domain.GetUserSession(ctx)
	if !ok || session.UserID == "" {
		return nil, errNoSession
	}
	return session, nil
}

// newSetting arma el setting por defecto del usuario de la sesión: idioma "es" y su cuenta más antigua de
// la agencia.
func (app *BondUserSettingApplication) newSetting(ctx context.Context, session *session_domain.UserSession) (*bondUserSetting_domain.BondUserSetting, error) {
	accountID, err := app.FirstAccount.FirstAccountID(ctx, session.UserID, session.AgencyID)
	if err != nil {
		return nil, err
	}
	return bondUserSetting_domain.NewBondUserSetting(session.UserID, session.AgencyID, accountID), nil
}

// upsertField guarda en el setting del usuario de la sesión el campo field con el valor que le pone set; si
// no tiene setting, lo crea con los valores por defecto. La cuenta más antigua solo se busca si hay que
// crearlo.
func (app *BondUserSettingApplication) upsertField(ctx context.Context, session *session_domain.UserSession, field bondUserSetting_ports.SettingField, set func(*bondUserSetting_domain.BondUserSetting)) (*bondUserSetting_domain.BondUserSetting, error) {
	existing, err := app.Repository.FindByUser(ctx, session.UserID)
	if err != nil {
		return nil, err
	}

	s := bondUserSetting_domain.NewBondUserSetting(session.UserID, session.AgencyID, nil)
	if existing == nil {
		if s, err = app.newSetting(ctx, session); err != nil {
			return nil, err
		}
	}
	set(s)
	return app.Repository.UpsertField(ctx, s, field)
}
