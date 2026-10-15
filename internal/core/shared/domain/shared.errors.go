package shared_domain

import "errors"

// Sentinels del dominio compartido. Los dominios que los reexponen delegan en MapToHTTPStatus.
var (
	// 400
	ErrAccountWithdrawAmountInvalid = errors.New("ERR_ACCOUNT_WITHDRAW_AMOUNT_INVALID")
	ErrAccountDepositAmountInvalid  = errors.New("ERR_ACCOUNT_DEPOSIT_AMOUNT_INVALID")
	// 404
	ErrAccountNotFound = errors.New("ERR_ACCOUNT_NOT_FOUND")
	// 422
	ErrAccountInactive                        = errors.New("ERR_ACCOUNT_INACTIVE")
	ErrAccountSuspended                       = errors.New("ERR_ACCOUNT_SUSPENDED")
	ErrAccountCancelled                       = errors.New("ERR_ACCOUNT_CANCELLED")
	ErrAccountStatus                          = errors.New("ERR_ACCOUNT_STATUS")
	ErrAccountWithdrawAmountMoreThanAvailable = errors.New("ERR_ACCOUNT_WITHDRAW_AMOUNT_MORE_THAN_AVAILABLE")
	// 502
	ErrExportFailed = errors.New("ERR_EXPORT_FAILED")
)
