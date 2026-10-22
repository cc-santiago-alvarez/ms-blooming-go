package bondUserSetting_domain

import (
	"errors"
	"net/http"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// MapToHTTPStatus devuelve el status HTTP de un error de bondUserSetting; lo que no es propio lo resuelve
// el dominio compartido.
func MapToHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	switch {
	case errors.Is(err, ErrInvalidLanguage):
		return http.StatusBadRequest
	case errors.Is(err, ErrBondUserSettingNotFound):
		return http.StatusNotFound
	}
	return shared_domain.MapToHTTPStatus(err)
}
