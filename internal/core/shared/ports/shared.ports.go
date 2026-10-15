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
