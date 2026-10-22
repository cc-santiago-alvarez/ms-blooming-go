package bondUserSetting_application

import (
	"context"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	bondUserSetting_ports "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/ports"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// SyncSelectedAccount cambia la cuenta destino de userID solo si tiene un setting vivo, e indica si lo
// cambió; nunca lo crea. La cuenta debe ser de userID: si no existe, está borrada o es de otro usuario, es
// ErrAccountNotFound. Un userId inválido es ErrInvalidID.
func (app *BondUserSettingApplication) SyncSelectedAccount(ctx context.Context, userID string, p bondUserSetting_domain.SetSelectedAccountParams) (bool, error) {
	if !shared_domain.IsValidID(userID) {
		return false, shared_domain.ErrInvalidID
	}
	p.Normalize()
	if err := p.Validation(); err != nil {
		return false, err
	}

	account, err := app.Balance.GetAccount(ctx, p.SelectedAccountID, userID)
	if err != nil {
		return false, err
	}
	if account == nil {
		return false, shared_domain.ErrAccountNotFound
	}

	s, err := app.Repository.FindByUser(ctx, userID)
	if err != nil || s == nil {
		return false, err
	}

	// con el documento existente, si lo borraron entretanto el alta choca con su _id y no crea otro
	s.SelectedAccountID = &account.ID
	if _, err := app.Repository.UpsertField(ctx, s, bondUserSetting_ports.FieldSelectedAccountID); err != nil {
		return false, err
	}
	return true, nil
}
