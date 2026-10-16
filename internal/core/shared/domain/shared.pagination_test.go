package shared_domain_test

import (
	"encoding/json"
	"testing"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Normalize deja Page en 1 o más y Limit entre 1 y 100, con 10 por defecto. CA-00.28
func TestPaginationNormalize(t *testing.T) {
	tests := []struct {
		name string
		in   shared_domain.Pagination
		want shared_domain.Pagination
	}{
		{name: "página cero y límite excesivo", in: shared_domain.Pagination{Page: 0, Limit: 500}, want: shared_domain.Pagination{Page: 1, Limit: 100}},
		{name: "valores negativos", in: shared_domain.Pagination{Page: -5, Limit: -1}, want: shared_domain.Pagination{Page: 1, Limit: 10}},
		{name: "límite cero", in: shared_domain.Pagination{Page: 2, Limit: 0}, want: shared_domain.Pagination{Page: 2, Limit: 10}},
		{name: "límite máximo", in: shared_domain.Pagination{Page: 1, Limit: 100}, want: shared_domain.Pagination{Page: 1, Limit: 100}},
		{name: "uno más que el máximo", in: shared_domain.Pagination{Page: 1, Limit: 101}, want: shared_domain.Pagination{Page: 1, Limit: 100}},
		{name: "valores válidos", in: shared_domain.Pagination{Page: 3, Limit: 25}, want: shared_domain.Pagination{Page: 3, Limit: 25}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.in
			got.Normalize()
			if got != tt.want {
				t.Fatalf("Normalize(%+v) = %+v, se esperaba %+v", tt.in, got, tt.want)
			}
		})
	}
}

// Skip es (page-1)*limit. CA-00.28
func TestPaginationSkip(t *testing.T) {
	tests := []struct {
		in   shared_domain.Pagination
		want int64
	}{
		{in: shared_domain.Pagination{Page: 3, Limit: 10}, want: 20},
		{in: shared_domain.Pagination{Page: 1, Limit: 10}, want: 0},
		{in: shared_domain.Pagination{Page: 5, Limit: 100}, want: 400},
	}
	for _, tt := range tests {
		if got := tt.in.Skip(); got != tt.want {
			t.Errorf("%+v.Skip() = %d, se esperaba %d", tt.in, got, tt.want)
		}
	}
}

// NewPaginated calcula totalPages, nunca deja items en nil y usa las claves JSON del contrato.
// CA-00.39
func TestNewPaginated(t *testing.T) {
	page := shared_domain.Pagination{Page: 2, Limit: 10}

	tests := []struct {
		name           string
		total          int64
		pagination     shared_domain.Pagination
		wantTotalPages int
	}{
		{name: "última página incompleta", total: 25, pagination: page, wantTotalPages: 3},
		{name: "páginas completas", total: 20, pagination: page, wantTotalPages: 2},
		{name: "sin resultados", total: 0, pagination: page, wantTotalPages: 0},
		{name: "límite sin normalizar", total: 25, pagination: shared_domain.Pagination{Page: 1}, wantTotalPages: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shared_domain.NewPaginated([]string{"a"}, tt.total, tt.pagination)
			if got.TotalPages != tt.wantTotalPages || got.Total != tt.total ||
				got.Page != tt.pagination.Page || got.Limit != tt.pagination.Limit {
				t.Fatalf("NewPaginated(total %d, %+v) = %+v, se esperaban %d páginas", tt.total, tt.pagination, got, tt.wantTotalPages)
			}
		})
	}

	t.Run("JSON sin resultados", func(t *testing.T) {
		got := shared_domain.NewPaginated[string](nil, 0, shared_domain.Pagination{Page: 1, Limit: 10})
		if got.Items == nil {
			t.Fatal("items quedó en nil")
		}
		body, err := json.Marshal(got)
		if err != nil {
			t.Fatalf("json.Marshal() = %v", err)
		}
		want := `{"items":[],"total":0,"page":1,"limit":10,"totalPages":0}`
		if string(body) != want {
			t.Fatalf("JSON %s, se esperaba %s", body, want)
		}
	})
}
