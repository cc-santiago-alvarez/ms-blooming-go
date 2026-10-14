package handlers_shared

import (
	"log"
	"net/http"

	"github.com/codecraftkit/nexus"
)

// errInternal es lo único que recibe el cliente ante un error no mapeado (RF-00.7).
const errInternal = "ERR_INTERNAL"

// ResponseWithError responde el error de un caso de uso con el status que le asignó MapToHTTPStatus.
// Un 500 es un error no mapeado o de infraestructura: el detalle se loguea con método y ruta, sin
// headers ni body, y el cliente solo recibe ERR_INTERNAL (RF-00.7, constitución §4). Los sentinels
// mapeados a otros 5xx (por ejemplo ERR_EXPORT_FAILED → 502) se responden tal cual.
func ResponseWithError(w http.ResponseWriter, r *http.Request, status int, err error) {
	if err == nil || status == 0 || status == http.StatusInternalServerError {
		log.Printf("[handler] error interno en %s %s: %v", r.Method, r.URL.Path, err)
		_ = nexus.ResponseWithError(w, http.StatusInternalServerError, errInternal)
		return
	}

	_ = nexus.ResponseWithError(w, status, err.Error())
}
