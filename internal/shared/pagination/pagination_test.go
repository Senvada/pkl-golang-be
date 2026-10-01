package pagination_test

import (
	"net/http/httptest"
	"testing"

	"pkl-golang-be/internal/shared/pagination"

	"github.com/gofiber/fiber/v2"
)

func TestFromContext_Defaults(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		p := pagination.FromContext(c)

		if p.Page != 1 {
			t.Errorf("Expected default page=1, got %d", p.Page)
		}
		if p.Limit != 10 {
			t.Errorf("Expected default limit=10, got %d", p.Limit)
		}
		if p.Order != "desc" {
			t.Errorf("Expected default order='desc', got '%s'", p.Order)
		}
		if p.SortBy != "created_at" {
			t.Errorf("Expected default sort_by='created_at', got '%s'", p.SortBy)
		}
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	_, _ = app.Test(req)
}

func TestFromContext_CustomAndBounds(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		p := pagination.FromContext(c)

		if p.Page != 3 {
			t.Errorf("Expected page=3, got %d", p.Page)
		}
		if p.Limit != 100 { // clamped from 500 to 100
			t.Errorf("Expected clamped limit=100, got %d", p.Limit)
		}
		if p.Search != "smk" {
			t.Errorf("Expected search='smk', got '%s'", p.Search)
		}
		if p.Order != "asc" {
			t.Errorf("Expected order='asc', got '%s'", p.Order)
		}

		meta := p.BuildMeta(45)
		if meta.TotalPages != 1 {
			t.Errorf("Expected total_pages=1 with limit 100, got %d", meta.TotalPages)
		}
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("GET", "/test?page=3&limit=500&search=smk&order=asc", nil)
	_, _ = app.Test(req)
}

func TestBuildMeta_Calculations(t *testing.T) {
	p := pagination.Params{
		Page:  2,
		Limit: 10,
	}

	meta := p.BuildMeta(25)
	if meta.TotalPages != 3 {
		t.Errorf("Expected total_pages=3 for 25 items with limit 10, got %d", meta.TotalPages)
	}
	if !meta.HasNext {
		t.Errorf("Expected has_next=true for page 2 of 3")
	}
	if !meta.HasPrev {
		t.Errorf("Expected has_prev=true for page 2 of 3")
	}
	if p.Offset() != 10 {
		t.Errorf("Expected offset=10 for page 2, got %d", p.Offset())
	}
}
