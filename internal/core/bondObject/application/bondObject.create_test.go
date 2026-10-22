package bondObject_application_test

import (
	"errors"
	"testing"

	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Un alta sin isPrivateBond ni inflectionPenaltyPercentage guarda isPrivateBond false y la penalidad en
// 100, y registra el history BOND_ADDED y la activity BOND_CREATED con el usuario y la agencia de la
// sesión. CA-02.4
func TestCreateDefaults(t *testing.T) {
	f := newFakes()
	params := createParams()

	got, err := f.app().Create(withSession(userID, agencyB), params)

	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.IsPrivateBond {
		t.Errorf("isPrivateBond true, se esperaba false por defecto")
	}
	if got.EarlyCashTerm.InflectionPenaltyPercentage != 100 {
		t.Errorf("inflectionPenaltyPercentage %v, se esperaba 100 por defecto", got.EarlyCashTerm.InflectionPenaltyPercentage)
	}
	if shared_domain.IDFormat(got.ID) != shared_domain.IDKindUUID {
		t.Errorf("_id %q, se esperaba un UUID v4", got.ID)
	}
	if len(f.repo.inserted) != 1 {
		t.Fatalf("InsertOne se llamó %d veces, se esperaba 1", len(f.repo.inserted))
	}
	if saved := f.repo.inserted[0]; saved.ID != got.ID || saved.IsPrivateBond || saved.EarlyCashTerm.InflectionPenaltyPercentage != 100 {
		t.Errorf("InsertOne recibió %+v, se esperaba el producto %s con los defaults", saved, got.ID)
	}

	if len(f.history.added) != 1 {
		t.Fatalf("History.Add se llamó %d veces, se esperaba 1", len(f.history.added))
	}
	if h := f.history.added[0]; h.TypeID != bondHistory_domain.HistoryBond || h.Text != shared_domain.TextBondAdded || h.RelatedID != got.ID {
		t.Errorf("History.Add recibió %+v, se esperaba typeId %q, text %q y relatedId %s", h, bondHistory_domain.HistoryBond, shared_domain.TextBondAdded, got.ID)
	}

	if len(f.activity.added) != 1 {
		t.Fatalf("Activity.Add se llamó %d veces, se esperaba 1", len(f.activity.added))
	}
	a := f.activity.added[0]
	if a.Description != shared_domain.TextBondCreated || a.RelatedID != got.ID {
		t.Errorf("Activity.Add recibió description %q y relatedId %q, se esperaba %q y %s", a.Description, a.RelatedID, shared_domain.TextBondCreated, got.ID)
	}
	if a.UserID != userID || a.AgencyID != agencyB {
		t.Errorf("la actividad tiene userId %q y agencyId %q, se esperaban los de la sesión %s y %s", a.UserID, a.AgencyID, userID, agencyB)
	}
	wantParams := map[string]any{
		"name":        params.Name,
		"description": params.Description,
		"amount":      params.Amount,
		"bondTypeId":  typeID,
	}
	if len(a.Params) != len(wantParams) {
		t.Errorf("params de la actividad %v, se esperaba %v", a.Params, wantParams)
	}
	for k, w := range wantParams {
		if a.Params[k] != w {
			t.Errorf("params[%s] = %v, se esperaba %v", k, a.Params[k], w)
		}
	}
}

// Un bondTypeId inexistente es ErrBondObjectInvalidBondType, y un inflectionPoint igual a
// maturityTimeInDays es ErrBondObjectInvalidInflectionPoint sin consultar el tipo; en los dos casos no se
// guarda nada ni se registra history ni activity. CA-02.5
func TestCreateRejected(t *testing.T) {
	missingType := createParams()
	missingType.BondTypeID = missingID

	inflectionAtMaturity := createParams()
	inflectionAtMaturity.MaturityTimeInDays = 365
	inflectionAtMaturity.EarlyCashTerm.InflectionPoint = 365

	tests := []struct {
		name          string
		params        bondObject_domain.CreateParams
		wantErr       error
		wantTypeReads int
	}{
		{name: "bondTypeId inexistente", params: missingType, wantErr: bondObject_domain.ErrBondObjectInvalidBondType, wantTypeReads: 1},
		{name: "inflectionPoint igual al plazo", params: inflectionAtMaturity, wantErr: bondObject_domain.ErrBondObjectInvalidInflectionPoint, wantTypeReads: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes()

			got, err := f.app().Create(withSession(userID, agencyA), tt.params)

			if !errors.Is(err, tt.wantErr) || got != nil {
				t.Fatalf("Create = %v, %v; se esperaba nil, %v", got, err, tt.wantErr)
			}
			if len(f.types.reads) != tt.wantTypeReads {
				t.Errorf("el tipo se consultó %d veces, se esperaban %d", len(f.types.reads), tt.wantTypeReads)
			}
			if len(f.repo.inserted) != 0 || len(f.history.added) != 0 || len(f.activity.added) != 0 {
				t.Fatalf("un alta rechazada guardó %d productos, %d history y %d activity",
					len(f.repo.inserted), len(f.history.added), len(f.activity.added))
			}
		})
	}
}
