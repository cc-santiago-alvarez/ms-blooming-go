package bondBound_domain

import (
	"strconv"
	"time"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// ObjectSnapshot son los datos del producto que copia un bono al comprarse.
type ObjectSnapshot struct {
	BondObjectID              string
	BondTypeID                string
	Key                       string
	Amount                    float64
	MaturityTimeInDays        int
	MaturityRewardsPercentage float64
	EarlyCashTerm             shared_domain.EarlyCashTerm
}

// Buyer es quien compra: el usuario y la agencia de la sesión.
type Buyer struct {
	UserID   string
	AgencyID string
}

// NewBoundFromObject arma el bono número seq del tipo del producto, iniciado, con la clave "<key>-<seq>", el
// token recibido y una copia de los términos del producto. Vence a la hora exacta de la compra,
// maturityTimeInDays días después; todas las fechas quedan en UTC.
func NewBoundFromObject(obj ObjectSnapshot, buyer Buyer, seq int, token string, now time.Time) *BondBound {
	at := now.UTC()
	keySeq := seq
	return &BondBound{
		ID:                        shared_domain.NewID(),
		BondObjectID:              obj.BondObjectID,
		BondTypeID:                obj.BondTypeID,
		Key:                       obj.Key + "-" + strconv.Itoa(seq),
		KeySeq:                    &keySeq,
		Token:                     token,
		UserID:                    buyer.UserID,
		AgencyID:                  buyer.AgencyID,
		StatusID:                  StatusInitiated,
		Amount:                    shared_domain.Round2(obj.Amount),
		MaturityTimeInDays:        obj.MaturityTimeInDays,
		MaturityRewardsPercentage: obj.MaturityRewardsPercentage,
		EarlyCashTerm:             obj.EarlyCashTerm,
		InitialDate:               at,
		MaturityDate:              at.Add(time.Duration(obj.MaturityTimeInDays) * 24 * time.Hour),
		CreatedAt:                 at,
		UpdatedAt:                 at,
		IsRemove:                  false,
	}
}
