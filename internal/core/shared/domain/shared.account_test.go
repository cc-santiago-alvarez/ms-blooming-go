package shared_domain_test

import (
	"testing"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// MaskAccountNumber oculta el número y deja los últimos 4 caracteres. CA-00.37
func TestMaskAccountNumber(t *testing.T) {
	tests := []struct {
		name   string
		number string
		want   string
	}{
		{name: "16 dígitos", number: "4000123412341234", want: "****************1234"},
		{name: "4 caracteres", number: "1234", want: "****************1234"},
		{name: "menos de 4", number: "12", want: "****************12"},
		{name: "vacío", number: "", want: "****************"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shared_domain.MaskAccountNumber(tt.number); got != tt.want {
				t.Fatalf("MaskAccountNumber(%q) = %q, se esperaba %q", tt.number, got, tt.want)
			}
		})
	}
}
