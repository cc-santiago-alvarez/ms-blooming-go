package shared_domain

// WithdrawParams son los datos de un retiro. UserID y AgencyID son los de la sesión.
type WithdrawParams struct {
	AccountID     string
	UserID        string
	AgencyID      string
	DescriptionID string
	Amount        float64
	Comment       string
}

// DepositParams son los datos de un depósito. UserID y AgencyID son los de la sesión.
type DepositParams struct {
	AccountID     string
	UserID        string
	AgencyID      string
	DescriptionID string
	Amount        float64
	Comment       string
}

// BalanceMovement resume un movimiento registrado: Amount lleva signo y Balance y Available son el
// saldo disponible después del movimiento.
type BalanceMovement struct {
	DetailID  string  `json:"detailId"`
	Reference string  `json:"reference"`
	Amount    float64 `json:"amount"`
	Balance   float64 `json:"balance"`
	Available float64 `json:"available"`
}
