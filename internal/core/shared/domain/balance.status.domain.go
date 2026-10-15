package shared_domain

// Estados de balanceAccount.
const (
	BalanceAccountStatusInactive  = "0"
	BalanceAccountStatusActive    = "1"
	BalanceAccountStatusSuspended = "2"
	BalanceAccountStatusCancelled = "3"
)

// operationTypeId de balanceDetail.
const (
	BalanceOperationTypeMD = "1" // resta
	BalanceOperationTypeMC = "2" // suma
)

// descriptionId de balanceDetail que usan los bonos.
const (
	BalanceDescriptionBondMC     = "6" // cobro de bonos
	BalanceDescriptionBondMD     = "7" // compra de bonos
	BalanceDescriptionBondSaleMC = "8" // venta de bonos
)

// typeId y text de balanceHistory; el typeId también es el categoryId de balanceOperation.
const (
	BalanceHistoryTypeDeposit  = "2"
	BalanceHistoryTypeWithdraw = "3"

	BalanceHistoryTextDeposit  = "BALANCE_ACCOUNT_DEPOSIT"
	BalanceHistoryTextWithdraw = "BALANCE_ACCOUNT_WITHDRAW"
)

// typeId de balanceOperation para un movimiento de una sola cuenta.
const BalanceOperationIndividual = "1"

// BalanceAccountStatusErr devuelve el error del estado de la cuenta, o nil si está activa.
func BalanceAccountStatusErr(statusID string) error {
	switch statusID {
	case BalanceAccountStatusActive:
		return nil
	case BalanceAccountStatusInactive:
		return ErrAccountInactive
	case BalanceAccountStatusSuspended:
		return ErrAccountSuspended
	case BalanceAccountStatusCancelled:
		return ErrAccountCancelled
	default:
		return ErrAccountStatus
	}
}
