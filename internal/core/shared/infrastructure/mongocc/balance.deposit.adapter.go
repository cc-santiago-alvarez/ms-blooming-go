package shared_mongocc_repository

import (
	"context"
	"math"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Deposit acredita p.Amount en la cuenta de p.UserID y registra el movimiento. No revisa el estado de
// la cuenta. Si ctx ya está en una transacción, se suma a esa; si no, abre una propia.
func (repo *BalanceMongoccRepository) Deposit(ctx context.Context, p shared_domain.DepositParams) (*shared_domain.BalanceMovement, error) {
	amount := shared_domain.Round2(p.Amount)
	// la negación también rechaza NaN
	if !(amount > 0) || math.IsInf(amount, 0) {
		return nil, shared_domain.ErrAccountDepositAmountInvalid
	}

	var mov *shared_domain.BalanceMovement
	err := repo.Tx.RunInTransaction(ctx, func(txCtx context.Context) error {
		filter := bson.M{"_id": p.AccountID, "userId": p.UserID, "isRemove": false}
		available, ok, err := repo.incAvailable(txCtx, filter, amount)
		if err != nil {
			return err
		}
		if !ok {
			return shared_domain.ErrAccountNotFound
		}

		mov, err = repo.writeMovement(txCtx, movement{
			accountID:       p.AccountID,
			userID:          p.UserID,
			agencyID:        p.AgencyID,
			descriptionID:   p.DescriptionID,
			comment:         p.Comment,
			operationTypeID: shared_domain.BalanceOperationTypeMC,
			historyTypeID:   shared_domain.BalanceHistoryTypeDeposit,
			historyText:     shared_domain.BalanceHistoryTextDeposit,
			amount:          amount,
		}, available)
		return err
	})
	if err != nil {
		return nil, err
	}
	return mov, nil
}
