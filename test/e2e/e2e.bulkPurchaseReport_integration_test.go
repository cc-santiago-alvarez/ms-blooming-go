//go:build integration

package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// reference es la referencia del movimiento de cada bono: 16 alfanuméricos.
var reference = regexp.MustCompile(`^[A-Za-z0-9]{16}$`)

// names son el nombre y el apellido de un perfil.
type names struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// named es la referencia por nombre a un tipo o a un producto.
type named struct {
	Name string `json:"name"`
}

// Un inversionista compra 3 bonos en una operación y los ve en su listado; el admin los ve en el listado de
// administración y los exporta en el reporte de comprados, que ms-exporter recibe con la moneda de la agencia
// y devuelve como Excel. RF-04.2, RF-07.1
func TestBulkPurchaseListingReport(t *testing.T) {
	setup(t)
	seed(t)

	var bulk struct {
		Items []struct {
			ID        string `json:"_id"`
			Key       string `json:"key"`
			Reference string `json:"reference"`
		} `json:"items"`
		Total            float64 `json:"total"`
		AccountAvailable float64 `json:"accountAvailable"`
	}
	call(t, http.MethodPost, "/bond-bounds/bulk-purchase", investorToken,
		map[string]any{"accountId": accountID, "bondObjectId": objectID, "quantity": 3}).expect(t, http.StatusCreated, &bulk)

	var ids, keys, references []string
	for _, item := range bulk.Items {
		ids = append(ids, item.ID)
		keys = append(keys, item.Key)
		references = append(references, item.Reference)
		if !reference.MatchString(item.Reference) {
			t.Errorf("reference de %s = %q, se esperaban 16 alfanuméricos", item.Key, item.Reference)
		}
	}
	wantKeys := []string{productKey + "-1", productKey + "-2", productKey + "-3"}
	if !slices.Equal(keys, wantKeys) {
		t.Fatalf("claves = %v, se esperaban %v", keys, wantKeys)
	}
	if len(slices.Compact(slices.Sorted(slices.Values(references)))) != 3 {
		t.Errorf("references = %v, se esperaban 3 distintas", references)
	}
	if bulk.Total != 3*price || bulk.AccountAvailable != initialAvailable-3*price {
		t.Fatalf("total %v y disponible %v, se esperaban %v y %v",
			bulk.Total, bulk.AccountAvailable, 3*price, initialAvailable-3*price)
	}
	assertAvailable(t, initialAvailable-3*price)
	if n, err := testDB.GetCollection("bondBound").CountDocuments(context.Background(), bson.M{"_id": bson.M{"$in": ids}}); err != nil || n != 3 {
		t.Fatalf("bonos guardados = %d, %v; se esperaban 3", n, err)
	}
	notifications.wait(t, notifyBulkPurchased, notifyWithdrawUser, notifyWithdrawAdmin)

	t.Run("listado del inversionista", func(t *testing.T) {
		var page struct {
			Items []struct {
				ID       string `json:"_id"`
				StatusID string `json:"statusId"`
			} `json:"items"`
			Total int `json:"total"`
		}
		call(t, http.MethodGet, "/bond-bounds/me", investorToken, nil).expect(t, http.StatusOK, &page)

		var got []string
		for _, item := range page.Items {
			got = append(got, item.ID)
			if item.StatusID != "1" {
				t.Errorf("statusId de %s = %q, se esperaba \"1\"", item.ID, item.StatusID)
			}
		}
		// los tres comparten createdAt
		if page.Total != 3 || !slices.Equal(slices.Sorted(slices.Values(got)), slices.Sorted(slices.Values(ids))) {
			t.Fatalf("listado = %v de %d, se esperaban %v", got, page.Total, ids)
		}
	})

	t.Run("listado de administración", func(t *testing.T) {
		var page struct {
			Items []struct {
				Key    string `json:"key"`
				Status struct {
					ID   string `json:"_id"`
					Name string `json:"name"`
				} `json:"status"`
				Profile    *names `json:"profile"`
				BondType   named  `json:"bondType"`
				BondObject named  `json:"bondObject"`
			} `json:"items"`
			Total int `json:"total"`
		}
		call(t, http.MethodGet, "/bond-bounds?"+dateRange(), adminToken, nil).expect(t, http.StatusOK, &page)

		var got []string
		for _, item := range page.Items {
			got = append(got, item.Key)
			switch {
			case item.Status.ID != "1" || item.Status.Name != "Iniciado":
				t.Errorf("status de %s = %+v, se esperaba 1 Iniciado", item.Key, item.Status)
			case item.Profile == nil || *item.Profile != (names{FirstName: "Ana", LastName: "Gómez"}):
				t.Errorf("profile de %s = %+v, se esperaba Ana Gómez", item.Key, item.Profile)
			case item.BondType.Name != typeName || item.BondObject.Name != productName:
				t.Errorf("tipo %q y producto %q de %s, se esperaban %q y %q",
					item.BondType.Name, item.BondObject.Name, item.Key, typeName, productName)
			}
		}
		if page.Total != 3 || !slices.Equal(slices.Sorted(slices.Values(got)), wantKeys) {
			t.Fatalf("listado = %v de %d, se esperaban %v", got, page.Total, wantKeys)
		}
	})

	t.Run("reporte de comprados", func(t *testing.T) {
		res := call(t, http.MethodGet, "/bond-bounds/reports/purchased?"+dateRange(), adminToken, nil)
		res.expect(t, http.StatusOK, nil)
		if !bytes.Equal(res.body, excelFile) {
			t.Errorf("archivo = %v, se esperaba %v", res.body, excelFile)
		}
		if ct := res.header.Get("Content-Type"); ct != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
			t.Errorf("Content-Type = %q, se esperaba el de xlsx", ct)
		}
		if cd := res.header.Get("Content-Disposition"); !strings.HasPrefix(cd, `attachment; filename="reporte-purchased-bonds_`) {
			t.Errorf("Content-Disposition = %q, se esperaba el adjunto reporte-purchased-bonds_<fecha>.xlsx", cd)
		}

		requests := exporter.received()
		if len(requests) != 1 {
			t.Fatalf("ms-exporter recibió %d requests, se esperaba 1", len(requests))
		}
		req := requests[0]
		if req.path != "/export/excel/report/purchasedBonds" || req.token != adminToken {
			t.Errorf("ms-exporter recibió %s con x-token %q, se esperaba /export/excel/report/purchasedBonds con %q",
				req.path, req.token, adminToken)
		}
		var payload struct {
			Currency   string `json:"currency"`
			BondBounds []struct {
				Key        string  `json:"key"`
				Status     string  `json:"status"`
				Amount     float64 `json:"amount"`
				Profile    *names  `json:"profile"`
				BondType   named   `json:"bondType"`
				BondObject named   `json:"bondObject"`
			} `json:"bondBounds"`
		}
		if err := json.Unmarshal(req.body, &payload); err != nil {
			t.Fatalf("body de ms-exporter %s inválido: %v", req.body, err)
		}
		if payload.Currency != currency {
			t.Errorf("currency = %q, se esperaba %q", payload.Currency, currency)
		}
		var got []string
		for _, row := range payload.BondBounds {
			got = append(got, row.Key)
			switch {
			case row.Status != "Iniciado" || row.Amount != price:
				t.Errorf("fila %s con status %q y amount %v, se esperaban Iniciado y %v", row.Key, row.Status, row.Amount, price)
			case row.Profile == nil || *row.Profile != (names{FirstName: "Ana", LastName: "Gómez"}):
				t.Errorf("profile de la fila %s = %+v, se esperaba Ana Gómez", row.Key, row.Profile)
			case row.BondType.Name != typeName || row.BondObject.Name != productName:
				t.Errorf("tipo %q y producto %q en la fila %s, se esperaban %q y %q",
					row.BondType.Name, row.BondObject.Name, row.Key, typeName, productName)
			}
		}
		if !slices.Equal(slices.Sorted(slices.Values(got)), wantKeys) {
			t.Fatalf("filas = %v, se esperaban %v", got, wantKeys)
		}
	})
}
