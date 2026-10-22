package bondUserSetting_application

import (
	"context"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	bondUserSetting_ports "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/ports"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// SetSelectedAccount cambia la cuenta destino del usuario de la sesión y devuelve el setting; si no tiene,
// lo crea. La cuenta debe ser suya: si no existe, está borrada o es de otro usuario, es ErrAccountNotFound.
func (app *BondUserSettingApplication) SetSelectedAccount(ctx context.Context, p bondUserSetting_domain.SetSelectedAccountParams) (*bondUserSetting_domain.BondUserSetting, error) {
	session, err := sessionUser(ctx)
	if err != nil {
		return nil, err
	}
	p.Normalize()
	if err := p.Validation(); err != nil {
		return nil, err
	}

	account, err := app.Balance.GetAccount(ctx, p.SelectedAccountID, session.UserID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, shared_domain.ErrAccountNotFound
	}

	s := bondUserSetting_domain.NewBondUserSetting(session.UserID, session.AgencyID, &account.ID)
	return app.Repository.UpsertField(ctx, s, bondUserSetting_ports.FieldSelectedAccountID)
}
