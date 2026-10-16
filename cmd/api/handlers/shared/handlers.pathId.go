package handlers_shared

import (
	"net/http"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// PathID lee el path param name y valida que sea un id UUID v4 o Meteor. Si no lo es, responde
// 400 ERR_INVALID_ID y devuelve false.
func PathID(w http.ResponseWriter, r *http.Request, name string) (string, bool) {
	id := r.PathValue(name)
	if !shared_domain.IsValidID(id) {
		ResponseWithError(w, r, http.StatusBadRequest, shared_domain.ErrInvalidID)
		return "", false
	}
	return id, true
}
