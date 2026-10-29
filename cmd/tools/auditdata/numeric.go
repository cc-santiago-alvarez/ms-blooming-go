package main

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// numericKind dice cómo lee Go el campo: con decimales, o entero truncado.
type numericKind int

const (
	kindFloat numericKind = iota
	kindInt
)

// numericField es un numérico que Meteor guardaba tal como llegaba del formulario.
type numericField struct {
	Path []string
	Kind numericKind
}

// name es la ruta del campo con puntos.
func (f numericField) name() string {
	return strings.Join(f.Path, ".")
}

// numericFields son los numéricos de bondObject, que la compra copia en bondBound.
var numericFields = []numericField{
	{Path: []string{"amount"}, Kind: kindFloat},
	{Path: []string{"maturityTimeInDays"}, Kind: kindInt},
	{Path: []string{"maturityRewardsPercentage"}, Kind: kindFloat},
	{Path: []string{"earlyCashTerm", "initialPenaltyPercentage"}, Kind: kindFloat},
	{Path: []string{"earlyCashTerm", "inflectionPoint"}, Kind: kindInt},
	{Path: []string{"earlyCashTerm", "inflectionPenaltyPercentage"}, Kind: kindFloat},
	{Path: []string{"earlyCashTerm", "initialDayForWithdraw"}, Kind: kindInt},
}

// numericAudit son los numéricos guardados como string de bondObject y bondBound.
type numericAudit struct {
	// Census cuenta, por "colección.campo", los documentos de cada tipo BSON.
	Census map[string]map[string]int
	// Plans son los documentos con strings canónicos, que -apply guarda como número.
	Plans []numericPlan
	// NonCanonical son los strings que no se corrigen solos.
	NonCanonical []docIssue
}

// numericPlan son los campos de un documento que -apply guarda como número.
type numericPlan struct {
	Collection string
	ID         bson.RawValue
	UpdatedAt  bson.RawValue
	Changes    []numericChange
}

// numericChange es un campo con su string y el número que lo reemplaza: int32 si es entero, si no double.
type numericChange struct {
	Field string
	Old   string
	New   any
}

// canonicalNumber es un número sin signo, espacios, exponente ni ceros de más: lo leen igual Go, Number,
// parseFloat y parseInt.
var canonicalNumber = regexp.MustCompile(`^(0|[1-9][0-9]*)(\.[0-9]*[1-9])?$`)

// parseCanonical devuelve el número de un string canónico, como int32 si es entero y entra, o como double. En
// un campo entero solo acepta enteros que entran en int32. Si no lo acepta, devuelve por qué.
func parseCanonical(s string, kind numericKind) (any, string) {
	if !canonicalNumber.MatchString(s) {
		return nil, nonCanonicalClass(s)
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsInf(v, 0) {
		return nil, "fuera de rango"
	}
	integral := v == math.Trunc(v)
	switch {
	case integral && v <= math.MaxInt32:
		return int32(v), ""
	case kind == kindInt && !integral:
		return nil, "decimal en un campo entero"
	case kind == kindInt:
		return nil, "fuera del rango de int32"
	}
	return v, ""
}

// nonCanonicalClass dice qué tiene un string que no es un número canónico, sin mostrarlo.
func nonCanonicalClass(s string) string {
	switch {
	case s == "":
		return "vacío"
	case strings.TrimSpace(s) != s:
		return "con espacios"
	}
	if _, err := strconv.ParseFloat(s, 64); err != nil {
		return "no numérico"
	}
	return "formato no canónico"
}

// visit cuenta los tipos de los numéricos del documento y planea los que están como string canónico.
func (n *numericAudit) visit(collection string, raw bson.Raw) {
	if n.Census == nil {
		n.Census = map[string]map[string]int{}
	}
	var plan *numericPlan
	for _, f := range numericFields {
		rv := raw.Lookup(f.Path...)
		key := collection + "." + f.name()
		if n.Census[key] == nil {
			n.Census[key] = map[string]int{}
		}
		n.Census[key][typeAlias(rv.Type)]++

		s, ok := rv.StringValueOK()
		if !ok {
			continue
		}
		v, problem := parseCanonical(s, f.Kind)
		if problem != "" {
			n.NonCanonical = append(n.NonCanonical, docIssue{
				Collection: collection,
				ID:         idString(raw.Lookup("_id")),
				Detail:     fmt.Sprintf("%s: %s, %s", f.name(), problem, plural(len(s), "carácter", "caracteres")),
			})
			continue
		}
		if plan == nil {
			plan = &numericPlan{
				Collection: collection,
				ID:         cloneValue(raw.Lookup("_id")),
				UpdatedAt:  cloneValue(raw.Lookup("updatedAt")),
			}
		}
		plan.Changes = append(plan.Changes, numericChange{Field: f.name(), Old: s, New: v})
	}
	if plan != nil {
		n.Plans = append(n.Plans, *plan)
	}
}

// count devuelve cuántos documentos de la colección tienen strings canónicos.
func (n numericAudit) count(collection string) int {
	total := 0
	for _, p := range n.Plans {
		if p.Collection == collection {
			total++
		}
	}
	return total
}

// applyNumeric guarda como número los strings canónicos, primero los de bondObject y después los de bondBound,
// para que una compra de Meteor que llegue en el medio copie números. Cada update exige que los campos sigan
// con el mismo string; no registra historial, porque el valor no cambia.
func (ap *applier) applyNumeric(ctx context.Context, plans []numericPlan) (applyResult, error) {
	res := applyResult{Fix: fixNumeric, Planned: len(plans)}
	for _, collection := range []string{bondObjectCollection, bondBoundCollection} {
		for _, p := range plans {
			if p.Collection != collection {
				continue
			}
			filter := bson.M{"_id": p.ID}
			set := bson.M{"updatedAt": ap.opts.runAt}
			change := backupChange{Kind: lineChange, Fix: fixNumeric, Collection: p.Collection, ID: p.ID}
			for _, ch := range p.Changes {
				filter[ch.Field] = ch.Old
				set[ch.Field] = ch.New
				change.Before = append(change.Before, bson.E{Key: ch.Field, Value: ch.Old})
				change.After = append(change.After, bson.E{Key: ch.Field, Value: ch.New})
			}
			change.withUpdatedAt(ap.lastUpdatedAt(p.Collection, p.ID, p.UpdatedAt), ap.opts.runAt)
			if err := ap.applyOne(ctx, &res, change, filter, bson.M{"$set": set}, nil); err != nil {
				return res, err
			}
		}
	}
	return res, nil
}
