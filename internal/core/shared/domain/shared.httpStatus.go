package shared_domain

import (
	"errors"
	"net/http"
)

// MapToHTTPStatus devuelve el status HTTP de un error del dominio compartido; lo no mapeado es 500.
func MapToHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	switch {
	case errors.Is(err, ErrInvalidID),
		errors.Is(err, ErrAccountWithdrawAmountInvalid),
		errors.Is(err, ErrAccountDepositAmountInvalid):
		return http.StatusBadRequest
	case errors.Is(err, ErrAccountNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrAccountInactive),
		errors.Is(err, ErrAccountSuspended),
		errors.Is(err, ErrAccountCancelled),
		errors.Is(err, ErrAccountStatus),
		errors.Is(err, ErrAccountWithdrawAmountMoreThanAvailable):
		return http.StatusUnprocessableEntity
	case errors.Is(err, ErrExportFailed):
		return http.StatusBadGateway
	}
	return http.StatusInternalServerError
}
