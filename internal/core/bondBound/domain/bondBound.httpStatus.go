package bondBound_domain

import (
	"errors"
	"net/http"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// MapToHTTPStatus devuelve el status HTTP de un error de bondBound; lo que no es propio lo resuelve el
// dominio compartido.
func MapToHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	switch {
	case errors.Is(err, ErrAccountIDRequired),
		errors.Is(err, ErrBondObjectIDRequired),
		errors.Is(err, ErrInvalidQuantity):
		return http.StatusBadRequest
	case errors.Is(err, ErrBondPrivateNotAuthorized):
		return http.StatusForbidden
	case errors.Is(err, ErrBondObjectNotFound),
		errors.Is(err, ErrBondBoundNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrAccountWithoutFunds),
		errors.Is(err, ErrAccountNotActive),
		errors.Is(err, ErrBondInactive),
		errors.Is(err, ErrInsufficientAmount),
		errors.Is(err, ErrBondNotAvailableByDate),
		errors.Is(err, ErrBondSaleEnded),
		errors.Is(err, ErrInvalidWithdrawDate),
		errors.Is(err, ErrBondInvalidTerms):
		return http.StatusUnprocessableEntity
	}
	return shared_domain.MapToHTTPStatus(err)
}
