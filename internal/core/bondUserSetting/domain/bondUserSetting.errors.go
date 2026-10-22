package bondUserSetting_domain

import "errors"

// Sentinels de bondUserSetting.
var (
	// 400
	ErrInvalidLanguage = errors.New("ERR_INVALID_LANGUAGE")
	// 404
	ErrBondUserSettingNotFound = errors.New("ERR_BOND_USER_SETTING_NOT_FOUND")
)
