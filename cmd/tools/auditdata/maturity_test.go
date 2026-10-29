package main

import (
	"slices"
	"testing"
	"time"
)

// fixedTime es una fecha fija de los tests.
var fixedTime = time.Date(2023, 7, 6, 22, 59, 58, 0, time.UTC)

// El corte es el inicio del día siguiente en UTC, como en la maduración de Go, aunque la fecha venga con otra
// zona.
func TestCutoffFor(t *testing.T) {
	cases := []struct {
		at   time.Time
		want time.Time
	}{
		{fixedTime, time.Date(2023, 7, 7, 0, 0, 0, 0, time.UTC)},
		{time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)},
		{time.Date(2026, 10, 13, 21, 0, 0, 0, time.FixedZone("", -5*3600)), time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)},
	}
	for _, tc := range cases {
		if got := cutoffFor(tc.at); !got.Equal(tc.want) {
			t.Errorf("cutoffFor(%s) = %s, se esperaba %s", tc.at, got, tc.want)
		}
	}
}

// -at vacío es la fecha actual; acepta RFC 3339 con cualquier zona y una fecha sola, en UTC.
func TestParseAt(t *testing.T) {
	now := time.Date(2026, 10, 29, 15, 4, 5, 0, time.UTC)
	cases := []struct {
		in   string
		want time.Time
	}{
		{"", now},
		{"2023-07-06", time.Date(2023, 7, 6, 0, 0, 0, 0, time.UTC)},
		{"2023-07-06T22:59:58Z", fixedTime},
		{"2023-07-06T17:59:58-05:00", fixedTime},
	}
	for _, tc := range cases {
		got, err := parseAt(tc.in, now)
		if err != nil || !got.Equal(tc.want) || got.Location() != time.UTC {
			t.Errorf("parseAt(%q) = %s, %v; se esperaba %s en UTC", tc.in, got, err, tc.want)
		}
	}
	if _, err := parseAt("06/07/2023", now); err == nil {
		t.Error("parseAt de una fecha con otro formato no dio error")
	}
}

// ms-feed descarta un bono si falta su profile, su bondType o su bondObject; los informa todos.
func TestFeedDrops(t *testing.T) {
	profiles := map[string]bool{"Hn4Wq8Ls2Dx6Vb9Ty": true}
	types := map[string]bool{"kQ3xT9mPz2LwA7bNc": true}
	objects := map[string]bool{"Bd9Xc4Vn7Mz2Lk5Jh": true}
	ok := boundRefs{ID: "a", UserID: "Hn4Wq8Ls2Dx6Vb9Ty", BondTypeID: "kQ3xT9mPz2LwA7bNc", BondObjectID: "Bd9Xc4Vn7Mz2Lk5Jh"}

	if got := feedDrops(ok, profiles, types, objects); len(got) != 0 {
		t.Errorf("un bono con sus tres referencias no se descarta: %v", got)
	}
	none := boundRefs{ID: "b"}
	if got := feedDrops(none, profiles, types, objects); !slices.Equal(got, []string{"sin profile", "sin bondType", "sin bondObject"}) {
		t.Errorf("un bono sin referencias = %v", got)
	}
	noObject := ok
	noObject.BondObjectID = "9e8d7c6b-5a4f-4e3d-b2c1-0a9b8c7d6e5f"
	if got := feedDrops(noObject, profiles, types, objects); !slices.Equal(got, []string{"sin bondObject"}) {
		t.Errorf("un bono sin su producto = %v", got)
	}
}
