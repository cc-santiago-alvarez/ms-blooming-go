package handlers_shared

import (
	"log"
	"net/http"

	"github.com/codecraftkit/nexus"
)

// errInternal es lo que recibe el cliente ante un error no mapeado.
const errInternal = "ERR_INTERNAL"

// ResponseWithError responde el error con su status. Un 500 se loguea y el cliente solo recibe
// ERR_INTERNAL.
func ResponseWithError(w http.ResponseWriter, r *http.Request, status int, err error) {
	if err == nil || status == 0 || status == http.StatusInternalServerError {
		log.Printf("[handler] error interno en %s %s: %v", r.Method, r.URL.Path, err)
		_ = nexus.ResponseWithError(w, http.StatusInternalServerError, errInternal)
		return
	}

	_ = nexus.ResponseWithError(w, status, err.Error())
}

// ResponseWithErrorDetail responde como ResponseWithError y agrega detail en errors.detail. Un 500 sale sin
// el detalle.
func ResponseWithErrorDetail(w http.ResponseWriter, r *http.Request, status int, err error, detail string) {
	if err == nil || status == 0 || status == http.StatusInternalServerError {
		ResponseWithError(w, r, status, err)
		return
	}

	// el mismo cuerpo que arma nexus.ResponseWithError, con el detalle
	_ = nexus.ResponseJsonWithError(w, status, &nexus.ErrorResponse{
		Code:     status,
		Message:  err.Error(),
		CodeName: "internal_server_error",
		Errors:   map[string]string{"error": err.Error(), "detail": detail},
	})
}
