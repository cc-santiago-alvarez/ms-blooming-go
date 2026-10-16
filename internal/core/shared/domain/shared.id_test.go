package shared_domain_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// IDFormat reconoce UUID v4 y Meteor de 17 caracteres; cualquier otro formato es inválido. CA-00.30
func TestIDFormat(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want shared_domain.IDKind
	}{
		{name: "UUID v4", id: "3f2b8c1e-9d4a-4f6b-8c2d-1a2b3c4d5e6f", want: shared_domain.IDKindUUID},
		{name: "UUID v4 en mayúsculas", id: "3F2B8C1E-9D4A-4F6B-AC2D-1A2B3C4D5E6F", want: shared_domain.IDKindUUID},
		{name: "UUID v4 variante 9", id: "0b6f8c2e-3d4a-4e5b-9c6d-7e8f9a0b1c2d", want: shared_domain.IDKindUUID},
		{name: "UUID v4 variante b", id: "6a5b4c3d-2e1f-4a0b-bc9d-7e6f5a4b3c2d", want: shared_domain.IDKindUUID},
		{name: "NewID", id: shared_domain.NewID(), want: shared_domain.IDKindUUID},

		{name: "Meteor", id: "tDr6NwSsDe4Lnccp8", want: shared_domain.IDKindMeteor},
		{name: "Meteor de otra colección", id: "oKDrhpKtbNs3degQe", want: shared_domain.IDKindMeteor},
		{name: "Meteor solo dígitos", id: "12345678901234567", want: shared_domain.IDKindMeteor},

		{name: "vacío", id: "", want: shared_domain.IDKindInvalid},
		{name: "texto corto", id: "abc", want: shared_domain.IDKindInvalid},
		{name: "ObjectId", id: "64b7f0c2a1e4d3b2c1a0f9e8", want: shared_domain.IDKindInvalid},
		{name: "17 caracteres con guion", id: "tDr6NwSsDe4Lncc-8", want: shared_domain.IDKindInvalid},
		{name: "UUID v1", id: "6ba7b810-9dad-11d1-80b4-00c04fd430c8", want: shared_domain.IDKindInvalid},
		{name: "UUID nulo", id: "00000000-0000-0000-0000-000000000000", want: shared_domain.IDKindInvalid},
		{name: "UUID v4 variante c", id: "3f2b8c1e-9d4a-4f6b-cc2d-1a2b3c4d5e6f", want: shared_domain.IDKindInvalid},
		{name: "UUID v4 variante 7", id: "3f2b8c1e-9d4a-4f6b-7c2d-1a2b3c4d5e6f", want: shared_domain.IDKindInvalid},
		{name: "UUID sin guiones", id: "3f2b8c1e9d4a4f6b8c2d1a2b3c4d5e6f", want: shared_domain.IDKindInvalid},
		{name: "UUID con llaves", id: "{3f2b8c1e-9d4a-4f6b-8c2d-1a2b3c4d5e6f}", want: shared_domain.IDKindInvalid},
		{name: "UUID con urn", id: "urn:uuid:3f2b8c1e-9d4a-4f6b-8c2d-1a2b3c4d5e6f", want: shared_domain.IDKindInvalid},
		{name: "UUID con salto de línea", id: "3f2b8c1e-9d4a-4f6b-8c2d-1a2b3c4d5e6f\n", want: shared_domain.IDKindInvalid},
		{name: "Meteor de 16", id: "tDr6NwSsDe4Lnccp", want: shared_domain.IDKindInvalid},
		{name: "Meteor de 18", id: "tDr6NwSsDe4Lnccp8x", want: shared_domain.IDKindInvalid},
		{name: "Meteor con ñ", id: "tDr6NwSsDe4Lnccpñ", want: shared_domain.IDKindInvalid},
		{name: "Meteor con espacio delante", id: " tDr6NwSsDe4Lnccp8", want: shared_domain.IDKindInvalid},
		{name: "Meteor con salto de línea", id: "tDr6NwSsDe4Lnccp8\n", want: shared_domain.IDKindInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shared_domain.IDFormat(tt.id); got != tt.want {
				t.Fatalf("IDFormat(%q) = %q, se esperaba %q", tt.id, got, tt.want)
			}
			wantValid := tt.want != shared_domain.IDKindInvalid
			if got := shared_domain.IsValidID(tt.id); got != wantValid {
				t.Fatalf("IsValidID(%q) = %v, se esperaba %v", tt.id, got, wantValid)
			}
		})
	}
}

// ErrInvalidID responde 400 ERR_INVALID_ID también cuando llega envuelto desde una validación. CA-00.31
func TestErrInvalidIDStatus(t *testing.T) {
	if got := shared_domain.ErrInvalidID.Error(); got != "ERR_INVALID_ID" {
		t.Fatalf("ErrInvalidID = %q, se esperaba ERR_INVALID_ID", got)
	}
	wrapped := fmt.Errorf("bondObjectId: %w", shared_domain.ErrInvalidID)
	if !errors.Is(wrapped, shared_domain.ErrInvalidID) {
		t.Fatal("el error envuelto no es ErrInvalidID")
	}
	if got := shared_domain.MapToHTTPStatus(wrapped); got != http.StatusBadRequest {
		t.Fatalf("MapToHTTPStatus(ErrInvalidID) = %d, se esperaba %d", got, http.StatusBadRequest)
	}
}
