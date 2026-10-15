package session_domain

// UserSession es la sesión que devuelve ms-oauth; los campos que no están aquí se ignoran.
type UserSession struct {
	UserID         string        `json:"userId"`
	Token          string        `json:"token"`
	AgencyID       string        `json:"agencyId"`
	Agency         SessionAgency `json:"agency"`
	CountryCode    string        `json:"countryCode"`
	FirstName      string        `json:"first_name"`
	LastName       string        `json:"last_name"`
	Email          string        `json:"email"`
	AdminBlooming  bool          `json:"adminBlooming"`
	SymbolCurrency string        `json:"symbolCurrency"`
	AccessUser     AccessUser    `json:"accessUser"`
}

// SessionAgency es la agencia del usuario; IsMain indica que no se filtra por agencia.
type SessionAgency struct {
	ID     string `json:"_id"`
	Name   string `json:"name"`
	IsMain bool   `json:"isMain"`
}

// AccessUser son los accesos de administración del usuario.
type AccessUser struct {
	AdminAccess []string        `json:"adminAccess"`
	IsAdmin     map[string]bool `json:"isAdmin"`
}
