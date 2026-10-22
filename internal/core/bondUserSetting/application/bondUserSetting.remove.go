package bondUserSetting_application

import (
	"context"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
)

// Remove borra de forma lógica el setting vivo del alcance de la sesión; si no existe, ya estaba borrado o
// es de otra agencia, devuelve ErrBondUserSettingNotFound. Un id inválido es ErrInvalidID.
func (app *BondUserSettingApplication) Remove(ctx context.Context, id string) error {
	s, err := app.findInScope(ctx, id)
	if err != nil {
		return err
	}

	ok, err := app.Repository.SoftDelete(ctx, s.ID)
	if err != nil {
		return err
	}
	if !ok {
		return bondUserSetting_domain.ErrBondUserSettingNotFound
	}
	return nil
}
