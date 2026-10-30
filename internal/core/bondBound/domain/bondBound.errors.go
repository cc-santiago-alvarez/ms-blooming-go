package bondBound_domain

import "errors"

// Sentinels de bondBound.
var (
	// 400
	ErrAccountIDRequired    = errors.New("ERR_ACCOUNT_ID_REQUIRED")
	ErrBondObjectIDRequired = errors.New("ERR_BOND_OBJECT_ID_REQUIRED")
	ErrInvalidQuantity      = errors.New("ERR_INVALID_QUANTITY")
	ErrInvalidDateFilter    = errors.New("ERR_INVALID_DATE_FILTER")
	ErrInvalidStatus        = errors.New("ERR_INVALID_STATUS")
	ErrInvalidSearch        = errors.New("ERR_INVALID_SEARCH")
	// 403
	ErrBondPrivateNotAuthorized = errors.New("ERR_BOND_PRIVATE_NOT_AUTHORIZED")
	// 404
	ErrBondObjectNotFound = errors.New("ERR_BOND_OBJECT_NOT_FOUND")
	ErrBondBoundNotFound  = errors.New("ERR_BOND_BOUND_NOT_FOUND")
	// 409
	ErrBondAlreadySold       = errors.New("ERR_BOND_ALREADY_SOLD")
	ErrBondAlreadyTook       = errors.New("ERR_BOND_ALREADY_TOOK")
	ErrBondStatusNotSellable = errors.New("ERR_BOND_STATUS_NOT_SELLABLE")
	ErrJobLocked             = errors.New("ERR_JOB_LOCKED")
	// 422
	ErrAccountWithoutFunds        = errors.New("ERR_ACCOUNT_WITHOUT_FUNDS")
	ErrAccountNotActive           = errors.New("ERR_ACCOUNT_NOT_ACTIVE")
	ErrBondInactive               = errors.New("ERR_BOND_INACTIVE")
	ErrInsufficientAmount         = errors.New("ERR_INSUFFICIENT_AMOUNT")
	ErrBondNotAvailableByDate     = errors.New("ERR_BOND_NOT_AVAILABLE_BY_DATE")
	ErrBondSaleEnded              = errors.New("ERR_BOND_SALE_ENDED")
	ErrInvalidWithdrawDate        = errors.New("ERR_INVALID_WITHDRAW_DATE")
	ErrBondInvalidTerms           = errors.New("ERR_BOND_INVALID_TERMS")
	ErrUserSettingAccountRequired = errors.New("ERR_USER_SETTING_ACCOUNT_REQUIRED")
	// 503
	ErrTradingDisabled = errors.New("ERR_TRADING_DISABLED")
)
