package bondUserSetting_ports

import (
	"context"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// SettingField es un campo del setting que el usuario puede cambiar.
type SettingField string

const (
	FieldLanguage            SettingField = "language"
	FieldSelectedAccountID   SettingField = "selectedAccountId"
	FieldSelectedBondBoundID SettingField = "selectedBondBoundId"
)

// BondUserSettingApplicationPorts son los casos de uso de bondUserSetting.
type BondUserSettingApplicationPorts interface {
	// GetMine devuelve el setting del usuario de la sesión con su cuenta, su perfil y el resumen de su
	// portafolio; si no tiene, lo crea con los valores por defecto.
	GetMine(ctx context.Context) (*bondUserSetting_domain.BondUserSettingDetail, error)
	// GetByUser devuelve el setting vivo de userID, o nil, nil si no tiene; nunca lo crea.
	GetByUser(ctx context.Context, userID string) (*bondUserSetting_domain.BondUserSetting, error)
	// SetSelectedAccount cambia la cuenta destino del usuario de la sesión y devuelve el setting; si no
	// tiene, lo crea. Si la cuenta no existe o no es suya, es ErrAccountNotFound.
	SetSelectedAccount(ctx context.Context, p bondUserSetting_domain.SetSelectedAccountParams) (*bondUserSetting_domain.BondUserSetting, error)
	// SetSelectedBondBound cambia el bono seleccionado del usuario de la sesión y devuelve el setting; si
	// no tiene, lo crea.
	SetSelectedBondBound(ctx context.Context, p bondUserSetting_domain.SetSelectedBondBoundParams) (*bondUserSetting_domain.BondUserSetting, error)
	// SetLanguage cambia el idioma del usuario de la sesión y devuelve el setting; si no tiene, lo crea. Un
	// idioma inválido es ErrInvalidLanguage.
	SetLanguage(ctx context.Context, p bondUserSetting_domain.SetLanguageParams) (*bondUserSetting_domain.BondUserSetting, error)
	// List devuelve la página de settings vivos del alcance de la sesión, por createdAt descendente.
	List(ctx context.Context, p shared_domain.Pagination) (shared_domain.Paginated[*bondUserSetting_domain.BondUserSetting], error)
	// GetByID devuelve el setting vivo, o ErrBondUserSettingNotFound. Un id inválido es ErrInvalidID.
	GetByID(ctx context.Context, id string) (*bondUserSetting_domain.BondUserSetting, error)
	// Remove borra el setting de forma lógica, o devuelve ErrBondUserSettingNotFound. Un id inválido es
	// ErrInvalidID.
	Remove(ctx context.Context, id string) error
	// SyncSelectedAccount cambia la cuenta destino de userID solo si tiene un setting vivo, e indica si lo
	// cambió. Un userId inválido es ErrInvalidID.
	SyncSelectedAccount(ctx context.Context, userID string, p bondUserSetting_domain.SetSelectedAccountParams) (bool, error)
}

// BondUserSettingMongoccRepositoryPorts es el repositorio de bondUserSetting. Un usuario tiene a lo sumo un
// setting vivo; mientras haya duplicados legacy, se usa el último modificado.
type BondUserSettingMongoccRepositoryPorts interface {
	// FindByUser devuelve el setting vivo de userID, o nil, nil si no tiene.
	FindByUser(ctx context.Context, userID string) (*bondUserSetting_domain.BondUserSetting, error)
	// UpsertDefault devuelve el setting vivo de s.UserID; si no tiene, guarda s de forma atómica. Llamadas
	// concurrentes crean un solo setting.
	UpsertDefault(ctx context.Context, s *bondUserSetting_domain.BondUserSetting) (*bondUserSetting_domain.BondUserSetting, error)
	// UpsertField guarda field con el valor que tiene en s, y updatedAt, en el setting vivo de s.UserID; si
	// no tiene, lo crea con los demás campos de s. Devuelve el setting actualizado.
	UpsertField(ctx context.Context, s *bondUserSetting_domain.BondUserSetting, field SettingField) (*bondUserSetting_domain.BondUserSetting, error)
	// FindPage devuelve la página p, ya normalizada, de los settings vivos por createdAt descendente.
	// agencyID nil no filtra por agencia.
	FindPage(ctx context.Context, agencyID *string, p shared_domain.Pagination) (shared_domain.Paginated[*bondUserSetting_domain.BondUserSetting], error)
	// FindOneByID devuelve nil, nil si el setting no existe o está borrado.
	FindOneByID(ctx context.Context, id string) (*bondUserSetting_domain.BondUserSetting, error)
	// SoftDelete marca el setting con isRemove true y actualiza updatedAt; devuelve false si no existe o ya
	// estaba borrado.
	SoftDelete(ctx context.Context, id string) (bool, error)
	// EnsureIndexes crea los índices de la colección; es idempotente.
	EnsureIndexes(ctx context.Context) error
}

// BoundSummaryPort resume en modo solo lectura los bonos de un usuario.
type BoundSummaryPort interface {
	// BoundSummary suma amount de los bonos no borrados de userID en agencyID, por estado.
	BoundSummary(ctx context.Context, userID, agencyID string) (bondUserSetting_domain.BoundSummary, error)
}

// FirstAccountPort lee en modo solo lectura las cuentas de saldo de un usuario.
type FirstAccountPort interface {
	// FirstAccountID devuelve el _id de la cuenta no borrada más antigua de userID en agencyID, o nil si no
	// tiene.
	FirstAccountID(ctx context.Context, userID, agencyID string) (*string, error)
}
