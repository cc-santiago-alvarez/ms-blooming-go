package shared_mongocc_repository

import (
	"context"
	"errors"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const referenceLength = 16

// movement es lo que se registra de un retiro o depósito; amount lleva signo.
type movement struct {
	accountID       string
	userID          string
	agencyID        string
	descriptionID   string
	comment         string
	operationTypeID string
	historyTypeID   string
	historyText     string
	amount          float64
	effectiveDate   *string
}

// incAvailable suma delta a available y global de la cuenta que cumple filter y devuelve el
// available resultante. ok es false si ninguna cuenta cumple el filtro.
func (repo *BalanceMongoccRepository) incAvailable(ctx context.Context, filter bson.M, delta float64) (available float64, ok bool, err error) {
	update := bson.M{
		"$inc": bson.M{"available": delta, "global": delta},
		"$set": bson.M{"updatedAt": shared_domain.Now()},
	}
	opts := options.FindOneAndUpdate().
		SetReturnDocument(options.After).
		SetProjection(bson.M{"available": 1})

	var account struct {
		Available float64 `bson:"available"`
	}
	if err := repo.DB.FindOneAndUpdate(ctx, balanceAccountCollection, filter, update, opts).Decode(&account); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return account.Available, true, nil
}

// writeMovement inserta el detalle, el historial y la operación del movimiento. available es el
// saldo de la cuenta ya actualizado.
func (repo *BalanceMongoccRepository) writeMovement(ctx context.Context, m movement, available float64) (*shared_domain.BalanceMovement, error) {
	detail := &shared_domain.BalanceDetail{
		BalanceAccountID: m.accountID,
		OperationTypeID:  m.operationTypeID,
		Reference:        repo.Utils.RandomAlnum(referenceLength),
		DescriptionID:    m.descriptionID,
		Amount:           m.amount,
		Balance:          shared_domain.Round2(available),
		AgencyID:         m.agencyID,
		EffectiveDate:    m.effectiveDate,
		UserID:           m.userID,
		Comment:          m.comment,
	}
	detail.New()
	if _, err := repo.DB.InsertOne(ctx, balanceDetailCollection, detail); err != nil {
		return nil, err
	}

	history := &shared_domain.BalanceHistory{
		BalanceAccountID: m.accountID,
		UserID:           m.userID,
		TypeID:           m.historyTypeID,
		Text:             m.historyText,
		Params:           []string{detail.ID},
		AgencyID:         m.agencyID,
	}
	history.New()
	if _, err := repo.DB.InsertOne(ctx, balanceHistoryCollection, history); err != nil {
		return nil, err
	}

	operation := &shared_domain.BalanceOperation{
		AgencyID:   m.agencyID,
		UserID:     m.userID,
		TypeID:     shared_domain.BalanceOperationIndividual,
		CategoryID: m.historyTypeID,
		Details:    []string{m.accountID, detail.ID},
		Concept:    m.comment,
	}
	operation.New()
	if _, err := repo.DB.InsertOne(ctx, balanceOperationCollection, operation); err != nil {
		return nil, err
	}

	return &shared_domain.BalanceMovement{
		DetailID:  detail.ID,
		Reference: detail.Reference,
		Amount:    detail.Amount,
		Balance:   detail.Balance,
		Available: detail.Balance,
	}, nil
}
