package shared_domain_test

import (
	"regexp"
	"testing"
	"time"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

var uuidV4Pattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

// stamp son los campos que New() asigna a toda entidad.
type stamp struct {
	id        string
	createdAt time.Time
	updatedAt time.Time
	isRemove  bool
}

// New() de cada entidad asigna _id UUID v4, createdAt == updatedAt en UTC e isRemove false. CA-00.11
func TestNewEntities(t *testing.T) {
	constructors := []struct {
		name string
		new  func() stamp
	}{
		{name: "BalanceDetail", new: func() stamp {
			d := shared_domain.BalanceDetail{ID: "viejo", IsRemove: true}
			d.New()
			return stamp{d.ID, d.CreatedAt, d.UpdatedAt, d.IsRemove}
		}},
		{name: "BalanceHistory", new: func() stamp {
			h := shared_domain.BalanceHistory{ID: "viejo", IsRemove: true}
			h.New()
			return stamp{h.ID, h.CreatedAt, h.UpdatedAt, h.IsRemove}
		}},
		{name: "BalanceOperation", new: func() stamp {
			o := shared_domain.BalanceOperation{ID: "viejo", IsRemove: true}
			o.New()
			return stamp{o.ID, o.CreatedAt, o.UpdatedAt, o.IsRemove}
		}},
		{name: "NoticeRecord", new: func() stamp {
			r := shared_domain.NewNoticeRecord(shared_domain.Notice{Type: shared_domain.NoticeTypeBond})
			return stamp{r.ID, r.CreatedAt, r.UpdatedAt, r.IsRemove}
		}},
		{name: "ActivityRecord", new: func() stamp {
			r := shared_domain.NewActivityRecord(shared_domain.Activity{Description: shared_domain.TextBondBoundPurchased})
			return stamp{r.ID, r.CreatedAt, r.UpdatedAt, r.IsRemove}
		}},
	}
	for _, c := range constructors {
		t.Run(c.name, func(t *testing.T) {
			before := shared_domain.Now()
			first, second := c.new(), c.new()
			after := shared_domain.Now()

			for _, s := range []stamp{first, second} {
				if !uuidV4Pattern.MatchString(s.id) {
					t.Fatalf("_id %q no es un UUID v4", s.id)
				}
				if !s.createdAt.Equal(s.updatedAt) {
					t.Fatalf("createdAt %v y updatedAt %v son distintos", s.createdAt, s.updatedAt)
				}
				if s.createdAt.Location() != time.UTC || s.updatedAt.Location() != time.UTC {
					t.Fatalf("las fechas no están en UTC: %v", s.createdAt.Location())
				}
				if s.createdAt.Before(before) || s.createdAt.After(after) {
					t.Fatalf("createdAt %v fuera de [%v, %v]", s.createdAt, before, after)
				}
				if s.isRemove {
					t.Fatal("isRemove quedó en true")
				}
			}
			if first.id == second.id {
				t.Fatalf("dos entidades nuevas tienen el mismo _id %q", first.id)
			}
		})
	}

	t.Run("NewID", func(t *testing.T) {
		seen := make(map[string]bool)
		for range 1000 {
			id := shared_domain.NewID()
			if !uuidV4Pattern.MatchString(id) {
				t.Fatalf("NewID() = %q, no es un UUID v4", id)
			}
			if seen[id] {
				t.Fatalf("NewID() repitió %q", id)
			}
			seen[id] = true
		}
	})
}
