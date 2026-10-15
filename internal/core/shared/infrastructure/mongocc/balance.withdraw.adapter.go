package shared_mongocc_repository

import (
	"context"

	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Withdraw descuenta p.Amount de la cuenta y registra el movimiento. Si ctx ya está en una
// transacción, se suma a esa; si no, abre una propia.
func (repo *BalanceMongoccRepository) Withdraw(ctx context.Context, p shared_domain.WithdrawParams) (*shared_domain.BalanceMovement, error) {
	amount := shared_domain.Round2(p.Amount)
	effectiveDate := shared_domain.EffectiveDate(shared_domain.Now(), session_domain.UtcDiff(ctx))

	var mov *shared_domain.BalanceMovement
	err := repo.Tx.RunInTransaction(ctx, func(txCtx context.Context) error {
		account, err := repo.GetAccount(txCtx, p.AccountID, p.UserID)
		if err != nil {
			return err
		}
		if account == nil {
			return shared_domain.ErrAccountNotFound
		}
		if err := account.CanWithdraw(amount); err != nil {
			return err
		}

		// El filtro por available evita que dos retiros simultáneos dejen el saldo negativo.
		filter := bson.M{"_id": p.AccountID, "available": bson.M{"$gte": amount}}
		available, ok, err := repo.incAvailable(txCtx, filter, -amount)
		if err != nil {
			return err
		}
		if !ok {
			return shared_domain.ErrAccountWithdrawAmountMoreThanAvailable
		}

		mov, err = repo.writeMovement(txCtx, movement{
			accountID:       p.AccountID,
			userID:          p.UserID,
			agencyID:        p.AgencyID,
			descriptionID:   p.DescriptionID,
			comment:         p.Comment,
			operationTypeID: shared_domain.BalanceOperationTypeMD,
			historyTypeID:   shared_domain.BalanceHistoryTypeWithdraw,
			historyText:     shared_domain.BalanceHistoryTextWithdraw,
			amount:          -amount,
			effectiveDate:   &effectiveDate,
		}, available)
		return err
	})
	if err != nil {
		return nil, err
	}
	return mov, nil
}
