package response_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"pkl-golang-be/internal/shared/pagination"
	"pkl-golang-be/internal/shared/response"

	"github.com/gofiber/fiber/v2"
)

func TestSingleResponse(t *testing.T) {
	app := fiber.New()
	app.Get("/test-single", func(c *fiber.Ctx) error {
		type UserDTO struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		return response.Single(c, UserDTO{ID: "123", Name: "Ahmad"}, "User fetched")
	})

	req := httptest.NewRequest(http.MethodGet, "/test-single", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	var env response.Envelope
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if !env.Success {
		t.Errorf("Expected success=true, got %v", env.Success)
	}
	if env.Message != "User fetched" {
		t.Errorf("Expected message 'User fetched', got '%s'", env.Message)
	}
	if env.Meta.RequestID == "" {
		t.Errorf("Expected request_id to be populated")
	}
}

func TestPaginatedResponse_Normal(t *testing.T) {
	app := fiber.New()
	app.Get("/test-paginated", func(c *fiber.Ctx) error {
		items := []string{"item1", "item2", "item3"}
		meta := pagination.Meta{
			Page:       1,
			Limit:      10,
			TotalRows:  25,
			TotalPages: 3,
			HasNext:    true,
			HasPrev:    false,
		}
		return response.Paginated(c, items, meta, "Items list")
	})

	req := httptest.NewRequest(http.MethodGet, "/test-paginated", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	var env struct {
		Success bool     `json:"success"`
		Message string   `json:"message"`
		Data    []string `json:"data"`
		Meta    struct {
			Pagination pagination.Meta `json:"pagination"`
		} `json:"meta"`
	}

	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if !env.Success {
		t.Errorf("Expected success=true, got %v", env.Success)
	}
	if len(env.Data) != 3 {
		t.Errorf("Expected 3 items, got %d", len(env.Data))
	}
	if env.Meta.Pagination.TotalRows != 25 {
		t.Errorf("Expected total_rows 25, got %d", env.Meta.Pagination.TotalRows)
	}
	if !env.Meta.Pagination.HasNext {
		t.Errorf("Expected has_next=true")
	}
}

func TestPaginatedResponse_NilSliceEnsuresEmptyArray(t *testing.T) {
	app := fiber.New()
	app.Get("/test-empty", func(c *fiber.Ctx) error {
		var nilSlice []string
		meta := pagination.Meta{
			Page:       1,
			Limit:      10,
			TotalRows:  0,
			TotalPages: 1,
			HasNext:    false,
			HasPrev:    false,
		}
		return response.Paginated(c, nilSlice, meta, "Empty items")
	})

	req := httptest.NewRequest(http.MethodGet, "/test-empty", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	jsonStr := string(body)

	// Verify JSON literally contains `"data":[]` and NOT `"data":null`
	if !contains(jsonStr, `"data":[]`) {
		t.Errorf("Expected '\"data\":[]' in json, got: %s", jsonStr)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
