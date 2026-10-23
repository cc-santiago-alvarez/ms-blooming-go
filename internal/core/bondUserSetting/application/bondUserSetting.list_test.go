package bondUserSetting_application_test

import (
	"slices"
	"testing"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Un admin de una agencia no principal lista solo los settings vivos de su agencia, paginados y del más
// reciente al más antiguo; uno de la agencia principal los ve todos. page y limit en 0 se normalizan.
// CA-09.8
func TestList(t *testing.T) {
	listed := func(id, userID, agencyID string, day int, removed bool) *bondUserSetting_domain.BondUserSetting {
		s := storedSetting(id, userID, agencyID)
		s.CreatedAt = storedAt.AddDate(0, 0, day)
		s.IsRemove = removed
		return s
	}
	settings := []*bondUserSetting_domain.BondUserSetting{
		listed("settingB1AAAAAAAA", "userB1AAAAAAAAAAA", agencyB, 1, false),
		listed("settingB2AAAAAAAA", "userB2AAAAAAAAAAA", agencyB, 2, false),
		listed("settingB3AAAAAAAA", "userB3AAAAAAAAAAA", agencyB, 3, false),
		listed("settingB4AAAAAAAA", "userB4AAAAAAAAAAA", agencyB, 4, true),
		listed("settingA1AAAAAAAA", "userA1AAAAAAAAAAA", agencyA, 5, false),
	}

	tests := []struct {
		name         string
		isMain       bool
		p            shared_domain.Pagination
		wantAgency   string
		wantIDs      []string
		wantTotal    int64
		wantPages    int
		wantPaginate shared_domain.Pagination
	}{
		{name: "agencia no principal, primera página", p: shared_domain.Pagination{Page: 1, Limit: 2}, wantAgency: agencyB,
			wantIDs: []string{"settingB3AAAAAAAA", "settingB2AAAAAAAA"}, wantTotal: 3, wantPages: 2, wantPaginate: shared_domain.Pagination{Page: 1, Limit: 2}},
		{name: "agencia no principal, segunda página", p: shared_domain.Pagination{Page: 2, Limit: 2}, wantAgency: agencyB,
			wantIDs: []string{"settingB1AAAAAAAA"}, wantTotal: 3, wantPages: 2, wantPaginate: shared_domain.Pagination{Page: 2, Limit: 2}},
		{name: "agencia principal", isMain: true, p: shared_domain.Pagination{Page: 1, Limit: 10},
			wantIDs: []string{"settingA1AAAAAAAA", "settingB3AAAAAAAA", "settingB2AAAAAAAA", "settingB1AAAAAAAA"}, wantTotal: 4, wantPages: 1, wantPaginate: shared_domain.Pagination{Page: 1, Limit: 10}},
		{name: "page y limit en 0", p: shared_domain.Pagination{}, wantAgency: agencyB,
			wantIDs: []string{"settingB3AAAAAAAA", "settingB2AAAAAAAA", "settingB1AAAAAAAA"}, wantTotal: 3, wantPages: 1, wantPaginate: shared_domain.Pagination{Page: 1, Limit: 10}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes(settings...)

			got, err := f.app().List(withAdmin(agencyB, tt.isMain), tt.p)

			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(f.repo.pages) != 1 {
				t.Fatalf("FindPage se llamó %d veces, se esperaba 1", len(f.repo.pages))
			}
			call := f.repo.pages[0]
			if tt.isMain && call.agencyID != nil {
				t.Errorf("FindPage recibió la agencia %s, se esperaba nil para la agencia principal", *call.agencyID)
			}
			if !tt.isMain && val(call.agencyID) != tt.wantAgency {
				t.Errorf("FindPage recibió la agencia %s, se esperaba %s", val(call.agencyID), tt.wantAgency)
			}
			if call.p != tt.wantPaginate {
				t.Errorf("FindPage recibió %+v, se esperaba %+v", call.p, tt.wantPaginate)
			}
			ids := []string{}
			for _, s := range got.Items {
				ids = append(ids, s.ID)
			}
			if !slices.Equal(ids, tt.wantIDs) {
				t.Errorf("List devolvió %v, se esperaba %v", ids, tt.wantIDs)
			}
			if got.Total != tt.wantTotal || got.TotalPages != tt.wantPages {
				t.Errorf("List devolvió total %d y %d páginas, se esperaban %d y %d", got.Total, got.TotalPages, tt.wantTotal, tt.wantPages)
			}
		})
	}
}
