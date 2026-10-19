package shared_domain

import (
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// EarlyCashTerm son las condiciones de retiro anticipado: las del producto en bondObject y la copia que
// guarda bondBound al comprar.
type EarlyCashTerm struct {
	InitialPenaltyPercentage    float64 `json:"initialPenaltyPercentage" bson:"initialPenaltyPercentage"`
	InflectionPoint             int     `json:"inflectionPoint" bson:"inflectionPoint"`
	InflectionPenaltyPercentage float64 `json:"inflectionPenaltyPercentage" bson:"inflectionPenaltyPercentage"`
	BlockWithdrawals            bool    `json:"blockWithdrawals" bson:"blockWithdrawals"`
	InitialDayForWithdraw       int     `json:"initialDayForWithdraw" bson:"initialDayForWithdraw"`
}

// earlyCashTermBSON lee los numéricos legacy, que pueden venir como string.
type earlyCashTermBSON struct {
	InitialPenaltyPercentage    FlexNumber `bson:"initialPenaltyPercentage"`
	InflectionPoint             FlexNumber `bson:"inflectionPoint"`
	InflectionPenaltyPercentage FlexNumber `bson:"inflectionPenaltyPercentage"`
	BlockWithdrawals            bool       `bson:"blockWithdrawals"`
	InitialDayForWithdraw       FlexNumber `bson:"initialDayForWithdraw"`
}

// UnmarshalBSONValue decodifica el subdocumento aceptando numéricos legacy; null o undefined lo dejan
// vacío. Los días se truncan a entero.
func (e *EarlyCashTerm) UnmarshalBSONValue(t byte, data []byte) error {
	switch bson.Type(t) {
	case bson.TypeNull, bson.TypeUndefined:
		*e = EarlyCashTerm{}
		return nil
	case bson.TypeEmbeddedDocument:
	default:
		return fmt.Errorf("earlyCashTerm: tipo BSON %s no soportado", bson.Type(t))
	}

	var raw earlyCashTermBSON
	if err := bson.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("earlyCashTerm: %w", err)
	}

	*e = EarlyCashTerm{
		InitialPenaltyPercentage:    float64(raw.InitialPenaltyPercentage),
		InflectionPoint:             raw.InflectionPoint.Int(),
		InflectionPenaltyPercentage: float64(raw.InflectionPenaltyPercentage),
		BlockWithdrawals:            raw.BlockWithdrawals,
		InitialDayForWithdraw:       raw.InitialDayForWithdraw.Int(),
	}
	return nil
}
