package shared_mongocc_repository

import (
	"github.com/codecraftkit/mongocc"
	utils_adapters "github.com/futurapps/ms-blooming-go/internal/core/shared/adapters/utils"
	shared_ports "github.com/futurapps/ms-blooming-go/internal/core/shared/ports"
)

// Colecciones de saldo del módulo legado. No se crean índices: se consultan por _id.
const (
	balanceAccountCollection   = "balanceAccount"
	balanceDetailCollection    = "balanceDetail"
	balanceHistoryCollection   = "balanceHistory"
	balanceOperationCollection = "balanceOperation"
)

// BalanceMongoccRepository implementa BalancePort sobre las colecciones de saldo.
type BalanceMongoccRepository struct {
	DB    *mongocc.MongoQueries
	Tx    shared_ports.TransactionRunnerPort
	Utils utils_adapters.UtilsPorts
}

var _ shared_ports.BalancePort = (*BalanceMongoccRepository)(nil)
