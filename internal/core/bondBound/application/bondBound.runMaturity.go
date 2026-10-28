package bondBound_application

import (
	"context"
	"log"
	"time"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Job de maduración: el nombre de su lock, lo que dura el lock y el tamaño de cada lote.
const (
	maturityJob       = "bond-maturity"
	maturityLockTTL   = 10 * time.Minute
	maturityBatchSize = 500
)

// RunMaturity pasa a madurados los bonos iniciados que vencen hasta el fin del día de now en UTC, de a lotes,
// con el lock del job tomado; si otra corrida lo tiene es ErrJobLocked. Cada bono cambia de estado y registra
// el historial del sistema en una transacción; si falla, se loguea, cuenta como fallido y la corrida sigue.
// Después avisa a cada dueño una vez, con todos sus bonos madurados. Si no puede leer un lote, avisa lo que
// ya maduró y devuelve el resultado parcial con el error. Una corrida empezada no se corta aunque ctx se
// cancele.
func (app *BondBoundApplication) RunMaturity(ctx context.Context, now time.Time) (bondBound_domain.MaturityResult, error) {
	ctx = context.WithoutCancel(ctx)
	start := time.Now()

	acquired, err := app.JobLock.Acquire(ctx, maturityJob, maturityLockTTL)
	if err != nil {
		return bondBound_domain.MaturityResult{}, err
	}
	if !acquired {
		return bondBound_domain.MaturityResult{}, bondBound_domain.ErrJobLocked
	}
	defer app.JobLock.Release(ctx, maturityJob)

	var result bondBound_domain.MaturityResult
	var matured []*bondBound_domain.BondBound
	var findErr error
	// inicio de mañana en UTC
	cutoff := shared_domain.StartOfDay(now, 0).AddDate(0, 0, 1)
	afterID := ""
	for {
		batch, err := app.Repository.FindMaturable(ctx, cutoff, maturityBatchSize, afterID)
		if err != nil {
			findErr = err
			log.Printf("[bondBound] no se pudo leer el lote de bonos por madurar después de %q: %v", afterID, err)
			break
		}
		for _, b := range batch {
			ok, err := app.mature(ctx, b, now)
			if err != nil {
				log.Printf("[bondBound] no se pudo madurar el bono %s: %v", b.ID, err)
				result.Failed++
				continue
			}
			if ok {
				result.Matured++
				matured = append(matured, b)
			}
		}
		if len(batch) < maturityBatchSize {
			break
		}
		afterID = batch[len(batch)-1].ID
	}

	groups := groupByUser(matured)
	result.Users = len(groups)
	app.notifyMaturity(ctx, groups)

	log.Printf("[bondBound] maduración: matured=%d users=%d failed=%d durationMs=%d",
		result.Matured, result.Users, result.Failed, time.Since(start).Milliseconds())
	return result, findErr
}

// mature pasa el bono de iniciado a madurado y registra el cambio en el historial, a nombre del sistema y en
// la agencia del bono, en una transacción. Devuelve false si el bono ya no estaba iniciado, sin soldDate ni
// tookDate; si algo falla no queda nada escrito.
func (app *BondBoundApplication) mature(ctx context.Context, b *bondBound_domain.BondBound, now time.Time) (bool, error) {
	var changed bool
	err := app.Tx.RunInTransaction(ctx, func(txCtx context.Context) error {
		var err error
		changed, err = app.Repository.UpdateStatusIf(txCtx, b.ID, bondBound_ports.StatusChange{
			From:      bondBound_domain.StatusInitiated,
			To:        bondBound_domain.StatusFinalized,
			MaturedAt: &now,
		})
		if err != nil || !changed {
			return err
		}

		return app.History.Add(session_domain.WithSystemAgency(txCtx, b.AgencyID), bondHistory_domain.AddParams{
			RelatedID: b.ID,
			TypeID:    bondHistory_domain.HistoryBondBound,
			Text:      shared_domain.TextBondBoundStatusChanged,
			Params:    []string{string(bondBound_domain.StatusInitiated), string(bondBound_domain.StatusFinalized)},
		})
	})
	if err != nil {
		return false, err
	}
	return changed, nil
}

// groupByUser agrupa los bonos por dueño, en el orden en que aparece cada uno.
func groupByUser(bounds []*bondBound_domain.BondBound) []maturedGroup {
	var groups []maturedGroup
	index := map[string]int{}
	for _, b := range bounds {
		i, ok := index[b.UserID]
		if !ok {
			i = len(groups)
			index[b.UserID] = i
			groups = append(groups, maturedGroup{userID: b.UserID})
		}
		groups[i].bounds = append(groups[i].bounds, b)
	}
	return groups
}
