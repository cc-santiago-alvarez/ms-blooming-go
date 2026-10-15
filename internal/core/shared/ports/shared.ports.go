package shared_ports

import (
	"context"

	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
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
