package bondObject_domain

import "errors"

// Sentinels de bondObject.
var (
	// 400
	ErrBondObjectNameRequired           = errors.New("ERR_BOND_OBJECT_NAME_REQUIRED")
	ErrBondObjectKeyRequired            = errors.New("ERR_BOND_OBJECT_KEY_REQUIRED")
	ErrBondObjectAgencyRequired         = errors.New("ERR_BOND_OBJECT_AGENCY_REQUIRED")
	ErrBondObjectInvalidAmount          = errors.New("ERR_BOND_OBJECT_INVALID_AMOUNT")
	ErrBondObjectInvalidMaturity        = errors.New("ERR_BOND_OBJECT_INVALID_MATURITY")
	ErrBondObjectInvalidInflectionPoint = errors.New("ERR_BOND_OBJECT_INVALID_INFLECTION_POINT")
	ErrBondObjectInvalidWithdrawDay     = errors.New("ERR_BOND_OBJECT_INVALID_WITHDRAW_DAY")
	ErrBondObjectInvalidPercentage      = errors.New("ERR_BOND_OBJECT_INVALID_PERCENTAGE")
	ErrBondObjectInvalidSaleWindow      = errors.New("ERR_BOND_OBJECT_INVALID_SALE_WINDOW")
	ErrBondObjectIDsRequired            = errors.New("ERR_BOND_OBJECT_IDS_REQUIRED")
	// 404
	ErrBondObjectNotFound = errors.New("ERR_BOND_OBJECT_NOT_FOUND")
	// 422
	ErrBondObjectInvalidBondType = errors.New("ERR_BOND_OBJECT_INVALID_BOND_TYPE")
)
