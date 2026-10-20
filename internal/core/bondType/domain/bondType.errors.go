package bondType_domain

import "errors"

// Sentinels de bondType.
var (
	// 400
	ErrBondTypeNameRequired   = errors.New("ERR_BOND_TYPE_NAME_REQUIRED")
	ErrBondTypeAgencyRequired = errors.New("ERR_BOND_TYPE_AGENCY_REQUIRED")
	// 404
	ErrBondTypeNotFound = errors.New("ERR_BOND_TYPE_NOT_FOUND")
	// 409
	ErrBondTypeHasBondObjects = errors.New("ERR_BOND_TYPE_HAS_BOND_OBJECTS")
)
