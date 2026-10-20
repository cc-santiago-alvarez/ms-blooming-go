package bondType_domain

import (
	"errors"
	"net/http"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// MapToHTTPStatus devuelve el status HTTP de un error de bondType; lo que no es propio lo resuelve el
// dominio compartido.
func MapToHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	switch {
	case errors.Is(err, ErrBondTypeNameRequired),
		errors.Is(err, ErrBondTypeAgencyRequired):
		return http.StatusBadRequest
	case errors.Is(err, ErrBondTypeNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrBondTypeHasBondObjects):
		return http.StatusConflict
	}
	return shared_domain.MapToHTTPStatus(err)
}
