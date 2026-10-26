package bondBound_application_test

import (
	"slices"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
)

// Statuses devuelve los seis estados en orden, con su nombre legacy. CA-06.6
func TestStatuses(t *testing.T) {
	want := []bondBound_domain.StatusRef{
		{ID: "1", Name: "Iniciado"},
		{ID: "2", Name: "Madurado"},
		{ID: "3", Name: "Bloqueado"},
		{ID: "4", Name: "Desligado"},
		{ID: "5", Name: "Vendido"},
		{ID: "6", Name: "Canjeado"},
	}

	got := newFakes().app().Statuses()

	if !slices.Equal(got, want) {
		t.Errorf("estados %+v, se esperaban %+v", got, want)
	}
}
