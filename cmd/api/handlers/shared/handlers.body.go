package handlers_shared

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// Errores de un body que no se puede leer.
var (
	ErrBodyRequired       = errors.New("BODY_REQUIRED")
	ErrInvalidRequestBody = errors.New("INVALID_REQUEST_BODY")
)

// maxBodyBytes es el tamaño máximo de un body JSON.
const maxBodyBytes = 1 << 20

// DecodeBody lee el body JSON en dst. Sin body responde 400 BODY_REQUIRED, y si no es un JSON válido para
// dst, 400 INVALID_REQUEST_BODY; en los dos casos devuelve false.
func DecodeBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	if r.Body == nil {
		ResponseWithError(w, r, http.StatusBadRequest, ErrBodyRequired)
		return false
	}

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	if err := decoder.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			ResponseWithError(w, r, http.StatusBadRequest, ErrBodyRequired)
		} else {
			ResponseWithError(w, r, http.StatusBadRequest, ErrInvalidRequestBody)
		}
		return false
	}
	// un segundo valor después del objeto también es un body inválido
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		ResponseWithError(w, r, http.StatusBadRequest, ErrInvalidRequestBody)
		return false
	}
	return true
}
