package bondHistory_domain

import (
	"errors"
	"net/http"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// MapToHTTPStatus devuelve el status HTTP de un error de bondHistory; lo que no es propio lo resuelve el
// dominio compartido.
func MapToHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if errors.Is(err, ErrBondHistoryInvalidParams) {
		return http.StatusInternalServerError
	}
	return shared_domain.MapToHTTPStatus(err)
}
