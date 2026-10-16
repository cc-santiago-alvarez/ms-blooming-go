package shared_ports

import (
	"context"

	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// SessionVerifier valida un token contra ms-oauth y devuelve la sesión.
type SessionVerifier interface {
	Verify(ctx context.Context, token string) (*session_domain.UserSession, error)
}

// TransactionRunnerPort ejecuta fn en una transacción de Mongo. Las escrituras de fn deben usar
// txCtx; con el ctx de afuera quedan fuera de la transacción.
type TransactionRunnerPort interface {
	RunInTransaction(ctx context.Context, fn func(txCtx context.Context) error) error
}

// BalancePort lee cuentas de saldo y registra retiros y depósitos. Las escrituras usan el ctx
// recibido, así que se suman a la transacción del llamador.
type BalancePort interface {
	// GetAccount devuelve nil, nil si la cuenta no existe o no es de userID.
	GetAccount(ctx context.Context, accountID, userID string) (*shared_domain.BalanceAccount, error)
	Withdraw(ctx context.Context, p shared_domain.WithdrawParams) (*shared_domain.BalanceMovement, error)
	Deposit(ctx context.Context, p shared_domain.DepositParams) (*shared_domain.BalanceMovement, error)
}

// NoticePort registra un aviso para un usuario. Es best-effort: un fallo se loguea y no se devuelve.
type NoticePort interface {
	Add(ctx context.Context, n shared_domain.Notice)
}

// ActivityPort registra una actividad de Blooming. Es best-effort: un fallo se loguea y no se devuelve.
type ActivityPort interface {
	Add(ctx context.Context, a shared_domain.Activity)
}

// NotificationPort envía una notificación por ms-notification. Es best-effort: un fallo se loguea
// y no se devuelve.
type NotificationPort interface {
	Send(ctx context.Context, group, notification string, params map[string]any)
}

// ProfileReader lee perfiles públicos de profile, sin datos sensibles.
type ProfileReader interface {
	// GetPublic devuelve los perfiles no borrados que existan entre ids; los demás no aparecen.
	GetPublic(ctx context.Context, ids ...string) ([]shared_domain.PublicProfile, error)
}

// AgencyReader lee datos de agency.
type AgencyReader interface {
	// SymbolCurrency devuelve el símbolo de moneda del país de la agencia, o "$" si no hay.
	SymbolCurrency(ctx context.Context, agencyID string) (string, error)
}

// ExporterPort genera reportes con ms-exporter.
type ExporterPort interface {
	// Excel devuelve el archivo del reporte; cualquier falla es ErrExportFailed.
	Excel(ctx context.Context, report string, payload any) ([]byte, error)
}
