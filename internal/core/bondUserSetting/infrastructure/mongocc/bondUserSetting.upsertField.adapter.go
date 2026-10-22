package bondUserSetting_mongocc_repository

import (
	"context"
	"fmt"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	bondUserSetting_ports "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/ports"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// UpsertField guarda field con el valor que tiene en s, y updatedAt, en el setting vivo de s.UserID; si no
// tiene, lo crea con los demás campos de s. Devuelve el setting actualizado.
func (repo *BondUserSettingMongoccRepository) UpsertField(ctx context.Context, s *bondUserSetting_domain.BondUserSetting, field bondUserSetting_ports.SettingField) (*bondUserSetting_domain.BondUserSetting, error) {
	value, err := fieldValue(s, field)
	if err != nil {
		return nil, err
	}

	// el campo no puede ir también en $setOnInsert
	onInsert := insertFields(s)
	delete(onInsert, string(field))

	update := bson.M{
		"$set":         bson.M{string(field): value, "updatedAt": shared_domain.Now()},
		"$setOnInsert": onInsert,
	}
	return repo.upsertLive(ctx, s.UserID, update)
}

// fieldValue devuelve el valor de field en s.
func fieldValue(s *bondUserSetting_domain.BondUserSetting, field bondUserSetting_ports.SettingField) (any, error) {
	switch field {
	case bondUserSetting_ports.FieldLanguage:
		return s.Language, nil
	case bondUserSetting_ports.FieldSelectedAccountID:
		return s.SelectedAccountID, nil
	case bondUserSetting_ports.FieldSelectedBondBoundID:
		return s.SelectedBondBoundID, nil
	}
	return nil, fmt.Errorf("bondUserSetting: campo %q no editable", field)
}
