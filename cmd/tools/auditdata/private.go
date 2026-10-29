package main

import (
	"context"
	"fmt"
	"math"

	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// privateAudit son los productos cuyo isPrivateBond no es booleano.
type privateAudit struct {
	// Missing cuenta los productos sin el campo, que Go y Meteor leen como públicos y no se tocan.
	Missing int
	// Auto son los que -apply guarda como booleano.
	Auto []privatePlan
	// Manual son los que quedan para revisión manual.
	Manual []docIssue
	// NoAuthorized son los productos privados, vivos y activos, sin una lista de authorizedUserIds: Meteor falla
	// al listar su tipo.
	NoAuthorized []docIssue
}

// privatePlan es un producto cuyo isPrivateBond -apply guarda como el booleano con el que ya lo leen Go y
// Meteor.
type privatePlan struct {
	ID        bson.RawValue
	Raw       bson.RawValue
	Target    bool
	AgencyID  string
	UpdatedAt bson.RawValue
	Detail    string
}

// visit clasifica el isPrivateBond del producto. obj es el producto decodificado por Go, o el valor cero si
// decodeErr no es nil.
func (p *privateAudit) visit(raw bson.Raw, obj bondObject_domain.BondObject, decodeErr error) {
	id := raw.Lookup("_id")
	rv := raw.Lookup("isPrivateBond")
	if rv.Type == 0 {
		p.Missing++
		return
	}
	private := obj.IsPrivateBond
	if b, ok := rv.BooleanOK(); ok {
		private = b
	}
	if private && listedByMeteor(raw) && !hasAuthorizedList(raw.Lookup("authorizedUserIds")) {
		p.NoAuthorized = append(p.NoAuthorized, docIssue{
			Collection: bondObjectCollection, ID: idString(id),
			Detail: "authorizedUserIds " + typeAlias(raw.Lookup("authorizedUserIds").Type),
		})
	}
	if rv.Type == bson.TypeBoolean {
		return
	}

	detail := describePrivate(rv, stringOf(raw.Lookup("bondTypeId")))
	manual := func(reason string) {
		p.Manual = append(p.Manual, docIssue{Collection: bondObjectCollection, ID: idString(id), Detail: detail + "; " + reason})
	}
	agency := ""
	if ids := obj.AgencyID.Normalize(); len(ids) > 0 {
		agency = ids[0]
	}
	switch {
	case decodeErr != nil:
		manual("Go no lo decodifica")
	case !autoPrivate(rv):
		manual("valor que Go y Meteor no leen igual o inesperado")
	case id.Type != bson.TypeString:
		manual("_id que no es string")
	case agency == "":
		manual("sin agencia para el historial")
	default:
		p.Auto = append(p.Auto, privatePlan{
			ID:        cloneValue(id),
			Raw:       cloneValue(rv),
			Target:    obj.IsPrivateBond,
			AgencyID:  agency,
			UpdatedAt: cloneValue(raw.Lookup("updatedAt")),
			Detail:    detail,
		})
	}
}

// listedByMeteor indica si el listado por tipo de Meteor lee el producto: pide isRemove false y active true.
func listedByMeteor(raw bson.Raw) bool {
	removed, okRemoved := raw.Lookup("isRemove").BooleanOK()
	active, okActive := raw.Lookup("active").BooleanOK()
	return okRemoved && !removed && okActive && active
}

// hasAuthorizedList indica si authorizedUserIds admite includes en JavaScript: un arreglo o un string.
func hasAuthorizedList(rv bson.RawValue) bool {
	return rv.Type == bson.TypeArray || rv.Type == bson.TypeString
}

// autoPrivate indica si el valor se normaliza solo: un string, null, un entero o un double finito, que Go y
// JavaScript leen igual y que el historial puede mostrar.
func autoPrivate(rv bson.RawValue) bool {
	switch rv.Type {
	case bson.TypeString, bson.TypeNull, bson.TypeInt32, bson.TypeInt64:
		return true
	case bson.TypeDouble:
		v, ok := rv.DoubleOK()
		return ok && !math.IsNaN(v) && !math.IsInf(v, 0)
	}
	return false
}

// describePrivate describe el valor sin mostrar un string: su tipo, su largo y si es el bondTypeId del
// producto.
func describePrivate(rv bson.RawValue, bondTypeID string) string {
	s, ok := rv.StringValueOK()
	if !ok {
		if rv.Type == bson.TypeInt32 || rv.Type == bson.TypeInt64 || rv.Type == bson.TypeDouble {
			return fmt.Sprintf("%s %v", typeAlias(rv.Type), historyValue(rv))
		}
		return typeAlias(rv.Type)
	}
	detail := "string de " + plural(len(s), "carácter", "caracteres")
	if s != "" && s == bondTypeID {
		detail += ", igual a su bondTypeId"
	}
	return detail
}

// historyValue es el valor guardado, como lo registra el historial.
func historyValue(rv bson.RawValue) any {
	switch rv.Type {
	case bson.TypeString:
		return stringOf(rv)
	case bson.TypeInt32:
		v, _ := rv.Int32OK()
		return v
	case bson.TypeInt64:
		v, _ := rv.Int64OK()
		return v
	case bson.TypeDouble:
		v, _ := rv.DoubleOK()
		return v
	}
	return nil
}

// applyPrivate guarda isPrivateBond como booleano en cada producto, con un BOND_UPDATED de sistema en la misma
// transacción. El update exige que el campo siga con el valor y el tipo que se leyeron.
func (ap *applier) applyPrivate(ctx context.Context, plans []privatePlan) (applyResult, error) {
	res := applyResult{Fix: fixPrivate, Planned: len(plans)}
	for _, p := range plans {
		id := stringOf(p.ID)
		h, err := systemHistory(bondHistory_domain.AddParams{
			RelatedID:        id,
			TypeID:           bondHistory_domain.HistoryBond,
			Text:             shared_domain.TextBondUpdated,
			Changes:          map[string]bondHistory_domain.ChangeEntry{"isPrivateBond": {Old: historyValue(p.Raw), New: p.Target}},
			EditedProperties: []string{"isPrivateBond"},
		}, p.AgencyID)
		if err != nil {
			res.Failed = append(res.Failed, docIssue{Collection: bondObjectCollection, ID: id, Detail: err.Error()})
			continue
		}
		change := backupChange{
			Kind: lineChange, Fix: fixPrivate, Collection: bondObjectCollection, ID: p.ID,
			Before:    bson.D{{Key: "isPrivateBond", Value: p.Raw}},
			After:     bson.D{{Key: "isPrivateBond", Value: p.Target}},
			HistoryID: h.ID,
		}
		change.withUpdatedAt(ap.lastUpdatedAt(bondObjectCollection, p.ID, p.UpdatedAt), ap.opts.runAt)
		filter := bson.M{"_id": p.ID, "isPrivateBond": exactMatch(p.Raw)}
		update := bson.M{"$set": bson.M{"isPrivateBond": p.Target, "updatedAt": ap.opts.runAt}}
		if err := ap.applyOne(ctx, &res, change, filter, update, h); err != nil {
			return res, err
		}
	}
	return res, nil
}
