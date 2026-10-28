package jobs_handlers

import (
	"net/http"
	"time"

	"github.com/codecraftkit/nexus"
	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// bondMaturityWriteTimeout es lo que puede tardar la respuesta de una corrida: nunca pasa del TTL de su lock.
const bondMaturityWriteTimeout = 10 * time.Minute

// BondMaturity madura los bonos que vencen hasta el fin del día en UTC y responde {matured, users, failed}.
// Si otra corrida tiene el lock responde 409 ERR_JOB_LOCKED.
func (handler *JobsHandler) BondMaturity(w http.ResponseWriter, r *http.Request) {
	// el WriteTimeout del servidor es más corto que una corrida larga
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(bondMaturityWriteTimeout))

	result, err := handler.bondBoundApplication.RunMaturity(r.Context(), shared_domain.Now())
	if err != nil {
		handlers_shared.ResponseWithError(w, r, bondBound_domain.MapToHTTPStatus(err), err)
		return
	}
	_ = nexus.ResponseWithJSON(w, http.StatusOK, result)
}
