package bondObject_domain

import (
	"maps"
	"slices"
	"time"
)

// Change es el valor anterior y el nuevo de un campo editado.
type Change struct {
	Old any `json:"old" bson:"old"`
	New any `json:"new" bson:"new"`
}

// BuildChanges compara el producto antes y después de una edición y devuelve los campos que cambiaron y
// sus nombres ordenados. Las fechas se comparan por instante, agencyId y authorizedUserIds como
// conjuntos y earlyCashTerm campo por campo, como "earlyCashTerm.<campo>". Sin cambios devuelve un mapa
// vacío y ninguna propiedad.
func BuildChanges(old, updated BondObject) (map[string]Change, []string) {
	changes := map[string]Change{}

	diff(changes, "name", old.Name, updated.Name)
	diff(changes, "description", old.Description, updated.Description)
	diff(changes, "key", old.Key, updated.Key)
	diff(changes, "bondTypeId", old.BondTypeID, updated.BondTypeID)
	diff(changes, "amount", old.Amount, updated.Amount)
	diff(changes, "isPrivateBond", old.IsPrivateBond, updated.IsPrivateBond)
	diff(changes, "maturityTimeInDays", old.MaturityTimeInDays, updated.MaturityTimeInDays)
	diff(changes, "maturityRewardsPercentage", old.MaturityRewardsPercentage, updated.MaturityRewardsPercentage)
	diff(changes, "active", old.Active, updated.Active)
	diff(changes, "available", old.Available, updated.Available)

	diffDate(changes, "saleInitDate", old.SaleInitDate, updated.SaleInitDate)
	diffDate(changes, "saleEndDate", old.SaleEndDate, updated.SaleEndDate)

	diffSet(changes, "agencyId", old.AgencyID, updated.AgencyID)
	diffSet(changes, "authorizedUserIds", old.AuthorizedUserIDs, updated.AuthorizedUserIDs)

	oldTerm, newTerm := old.EarlyCashTerm, updated.EarlyCashTerm
	diff(changes, "earlyCashTerm.initialPenaltyPercentage", oldTerm.InitialPenaltyPercentage, newTerm.InitialPenaltyPercentage)
	diff(changes, "earlyCashTerm.inflectionPoint", oldTerm.InflectionPoint, newTerm.InflectionPoint)
	diff(changes, "earlyCashTerm.inflectionPenaltyPercentage", oldTerm.InflectionPenaltyPercentage, newTerm.InflectionPenaltyPercentage)
	diff(changes, "earlyCashTerm.blockWithdrawals", oldTerm.BlockWithdrawals, newTerm.BlockWithdrawals)
	diff(changes, "earlyCashTerm.initialDayForWithdraw", oldTerm.InitialDayForWithdraw, newTerm.InitialDayForWithdraw)

	return changes, slices.Sorted(maps.Keys(changes))
}

// diff registra el campo si el valor cambió.
func diff[T comparable](changes map[string]Change, field string, old, updated T) {
	if old != updated {
		changes[field] = Change{Old: old, New: updated}
	}
}

// diffDate registra la fecha si cambió el instante; una fecha que no está queda en nil.
func diffDate(changes map[string]Change, field string, old, updated *time.Time) {
	if !sameInstant(old, updated) {
		changes[field] = Change{Old: dateValue(old), New: dateValue(updated)}
	}
}

// diffSet registra la lista completa si cambiaron sus elementos, sin importar el orden ni los repetidos.
func diffSet[S ~[]string](changes map[string]Change, field string, old, updated S) {
	if !sameSet(old, updated) {
		changes[field] = Change{Old: listValue(old), New: listValue(updated)}
	}
}

// sameInstant indica si las dos fechas faltan o son el mismo instante.
func sameInstant(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}

// sameSet indica si las dos listas tienen los mismos elementos.
func sameSet(a, b []string) bool {
	inA := make(map[string]bool, len(a))
	for _, v := range a {
		inA[v] = true
	}
	inB := make(map[string]bool, len(b))
	for _, v := range b {
		if !inA[v] {
			return false
		}
		inB[v] = true
	}
	return len(inA) == len(inB)
}

// dateValue devuelve la fecha, o un nil sin tipo si no está.
func dateValue(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

// listValue devuelve la lista como []string, con [] en lugar de nil.
func listValue[S ~[]string](ids S) []string {
	if ids == nil {
		return []string{}
	}
	return []string(ids)
}
