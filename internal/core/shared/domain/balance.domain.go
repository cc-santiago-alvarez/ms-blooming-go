package shared_domain

import "time"

// BalanceAccount es la parte de balanceAccount que lee el ms.
type BalanceAccount struct {
	ID        string  `json:"_id" bson:"_id"`
	UserID    string  `json:"userId" bson:"userId"`
	AgencyID  string  `json:"agencyId" bson:"agencyId"`
	StatusID  string  `json:"statusId" bson:"statusId"`
	Available float64 `json:"available" bson:"available"`
	Number    string  `json:"number" bson:"number"`
}

// CanWithdraw valida, en el orden del legado, que la cuenta esté activa y que amount sea positivo y
// no supere el disponible.
func (a *BalanceAccount) CanWithdraw(amount float64) error {
	if err := BalanceAccountStatusErr(a.StatusID); err != nil {
		return err
	}
	// la negación también rechaza NaN
	if !(amount > 0) {
		return ErrAccountWithdrawAmountInvalid
	}
	if amount > a.Available {
		return ErrAccountWithdrawAmountMoreThanAvailable
	}
	return nil
}

// BalanceDetail es un movimiento de balanceDetail.
type BalanceDetail struct {
	ID               string    `json:"_id" bson:"_id"`
	BalanceAccountID string    `json:"balanceAccountId" bson:"balanceAccountId"`
	OperationTypeID  string    `json:"operationTypeId" bson:"operationTypeId"`
	Reference        string    `json:"reference" bson:"reference"`
	DescriptionID    string    `json:"descriptionId" bson:"descriptionId"`
	Amount           float64   `json:"amount" bson:"amount"`
	Balance          float64   `json:"balance" bson:"balance"`
	AgencyID         string    `json:"agencyId" bson:"agencyId"`
	EffectiveDate    *string   `json:"effectiveDate" bson:"effectiveDate"`
	UserID           string    `json:"userId" bson:"userId"`
	Comment          string    `json:"comment" bson:"comment"`
	CreatedAt        time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt" bson:"updatedAt"`
	IsRemove         bool      `json:"isRemove" bson:"isRemove"`
}

func (d *BalanceDetail) New() {
	d.ID = NewID()
	d.CreatedAt = Now()
	d.UpdatedAt = d.CreatedAt
	d.IsRemove = false
}

// BalanceHistory es una entrada de balanceHistory.
type BalanceHistory struct {
	ID               string    `json:"_id" bson:"_id"`
	BalanceAccountID string    `json:"balanceAccountId" bson:"balanceAccountId"`
	UserID           string    `json:"userId" bson:"userId"`
	TypeID           string    `json:"typeId" bson:"typeId"`
	Text             string    `json:"text" bson:"text"`
	Params           []string  `json:"params" bson:"params"`
	AgencyID         string    `json:"agencyId" bson:"agencyId"`
	CreatedAt        time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt" bson:"updatedAt"`
	IsRemove         bool      `json:"isRemove" bson:"isRemove"`
}

func (h *BalanceHistory) New() {
	h.ID = NewID()
	h.CreatedAt = Now()
	h.UpdatedAt = h.CreatedAt
	h.IsRemove = false
}

// BalanceOperation es una entrada de balanceOperation.
type BalanceOperation struct {
	ID         string    `json:"_id" bson:"_id"`
	AgencyID   string    `json:"agencyId" bson:"agencyId"`
	UserID     string    `json:"userId" bson:"userId"`
	TypeID     string    `json:"typeId" bson:"typeId"`
	CategoryID string    `json:"categoryId" bson:"categoryId"`
	Details    []string  `json:"details" bson:"details"`
	ExtraInfo  any       `json:"extraInfo" bson:"extraInfo"`
	Concept    string    `json:"concept" bson:"concept"`
	CreatedAt  time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt" bson:"updatedAt"`
	IsRemove   bool      `json:"isRemove" bson:"isRemove"`
}

func (o *BalanceOperation) New() {
	o.ID = NewID()
	o.CreatedAt = Now()
	o.UpdatedAt = o.CreatedAt
	o.IsRemove = false
}
