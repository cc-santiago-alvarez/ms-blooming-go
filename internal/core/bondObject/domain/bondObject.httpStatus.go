package bondObject_domain

import (
	"errors"
	"net/http"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// MapToHTTPStatus devuelve el status HTTP de un error de bondObject; lo que no es propio lo resuelve el
// dominio compartido.
func MapToHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	switch {
	case errors.Is(err, ErrBondObjectNameRequired),
		errors.Is(err, ErrBondObjectKeyRequired),
		errors.Is(err, ErrBondObjectAgencyRequired),
		errors.Is(err, ErrBondObjectInvalidAmount),
		errors.Is(err, ErrBondObjectInvalidMaturity),
		errors.Is(err, ErrBondObjectInvalidInflectionPoint),
		errors.Is(err, ErrBondObjectInvalidWithdrawDay),
		errors.Is(err, ErrBondObjectInvalidPercentage),
		errors.Is(err, ErrBondObjectInvalidSaleWindow),
		errors.Is(err, ErrBondObjectIDsRequired):
		return http.StatusBadRequest
	case errors.Is(err, ErrBondObjectNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrBondObjectInvalidBondType):
		return http.StatusUnprocessableEntity
	}
	return shared_domain.MapToHTTPStatus(err)
}
