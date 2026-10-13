package role_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/infra/database"
	"pkl-golang-be/internal/middleware"
	"pkl-golang-be/internal/slices/auth"
	"pkl-golang-be/internal/slices/role"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type testContext struct {
	app     *fiber.App
	cfg     *config.Config
	db      *gorm.DB
	service role.Service
}

func setupTestContext(t *testing.T) *testContext {
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.CustomErrorHandler,
	})

	api := app.Group("/api/v1")

	roleSvc := role.NewService(db, cfg)
	roleHandler := role.NewHandler(roleSvc, cfg)
	role.RegisterRoutes(api, roleHandler, cfg)

	return &testContext{
		app:     app,
		cfg:     cfg,
		db:      db,
		service: roleSvc,
	}
}

func getAdminToken(t *testing.T, tc *testContext) string {
	var adminRole auth.Role
	err := tc.db.Where("name = ?", "Admin").First(&adminRole).Error
	if err != nil {
		adminRole = auth.Role{
			Name:        "Admin",
			Color:       "#EF4444",
			Permissions: datatypes.JSON([]byte(`{"admin": true}`)),
		}
		if err := tc.db.Create(&adminRole).Error; err != nil {
			t.Fatalf("Failed to ensure admin role: %v", err)
		}
	}

	testUser := &auth.User{
		ID:     uuid.New(),
		Email:  "test_role_admin@example.com",
		RoleID: adminRole.ID,
	}

	token, _, err := auth.GenerateAccessToken(testUser, "Admin", tc.cfg.JWTSecret)
	if err != nil {
		t.Fatalf("Failed to generate test admin token: %v", err)
	}
	return token
}

func makeRequest(app *fiber.App, method, path, token string, body any) (*http.Response, map[string]any, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := app.Test(req, 5000)
	if err != nil {
		return nil, nil, err
	}

	respBody, _ := io.ReadAll(resp.Body)
	var parsed map[string]any
	_ = json.Unmarshal(respBody, &parsed)

	return resp, parsed, nil
}

func TestRoleLifecycle(t *testing.T) {
	tc := setupTestContext(t)
	token := getAdminToken(t, tc)

	// 1. Get all roles
	resp, parsed, err := makeRequest(tc.app, "GET", "/api/v1/roles", token, nil)
	if err != nil {
		t.Fatalf("Failed to get roles: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK on GET /roles, got %d. Body: %v", resp.StatusCode, parsed)
	}

	// 2. Create a new test role
	uniqueName := "RoleTest_" + uuid.New().String()[:8]
	createPayload := map[string]any{
		"name":  uniqueName,
		"color": "#10B981",
		"permissions": map[string]any{
			"users": map[string]any{
				"read":   true,
				"create": false,
			},
		},
	}

	resp, parsed, err = makeRequest(tc.app, "POST", "/api/v1/roles", token, createPayload)
	if err != nil {
		t.Fatalf("Failed to create role: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201 Created on POST /roles, got %d. Body: %v", resp.StatusCode, parsed)
	}

	dataMap, ok := parsed["data"].(map[string]any)
	if !ok {
		t.Fatalf("Expected data map in response, got %v", parsed)
	}
	createdIDStr, ok := dataMap["id"].(string)
	if !ok || createdIDStr == "" {
		t.Fatalf("Expected created role id in response, got %v", dataMap)
	}
	createdID, err := uuid.Parse(createdIDStr)
	if err != nil {
		t.Fatalf("Invalid uuid: %v", err)
	}

	// 3. Duplicate name check
	resp, _, err = makeRequest(tc.app, "POST", "/api/v1/roles", token, createPayload)
	if err != nil {
		t.Fatalf("Failed to post duplicate: %v", err)
	}
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("Expected 409 Conflict on duplicate role name, got %d", resp.StatusCode)
	}

	// 4. Get by ID
	resp, parsed, err = makeRequest(tc.app, "GET", "/api/v1/roles/"+createdIDStr, token, nil)
	if err != nil {
		t.Fatalf("Failed to get role by id: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK on GET /roles/:id, got %d", resp.StatusCode)
	}

	// 5. Update role
	updatedName := uniqueName + "_Updated"
	updatePayload := map[string]any{
		"name":  updatedName,
		"color": "#3B82F6",
		"permissions": map[string]any{
			"users": map[string]any{
				"read":   true,
				"create": true,
			},
		},
	}
	resp, parsed, err = makeRequest(tc.app, "PUT", "/api/v1/roles/"+createdIDStr, token, updatePayload)
	if err != nil {
		t.Fatalf("Failed to update role: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK on PUT /roles/:id, got %d. Body: %v", resp.StatusCode, parsed)
	}

	// 6. Delete role
	resp, _, err = makeRequest(tc.app, "DELETE", "/api/v1/roles/"+createdIDStr, token, nil)
	if err != nil {
		t.Fatalf("Failed to delete role: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK on DELETE /roles/:id, got %d", resp.StatusCode)
	}

	// Clean up from database
	_ = tc.db.Unscoped().Where("id = ?", createdID).Delete(&role.Role{})
}

func TestAdminRoleProtection(t *testing.T) {
	tc := setupTestContext(t)
	token := getAdminToken(t, tc)

	var adminRole auth.Role
	if err := tc.db.Where("name = ?", "Admin").First(&adminRole).Error; err != nil {
		t.Fatalf("Admin role not found: %v", err)
	}

	// Attempt to delete Admin role -> should be Forbidden 403
	resp, parsed, err := makeRequest(tc.app, "DELETE", "/api/v1/roles/"+adminRole.ID.String(), token, nil)
	if err != nil {
		t.Fatalf("Failed to request delete Admin: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden when deleting Admin role, got %d. Body: %v", resp.StatusCode, parsed)
	}

	// Attempt to rename Admin role -> should be Forbidden 403
	renamePayload := map[string]any{
		"name": "BukanAdminLagi",
	}
	resp, parsed, err = makeRequest(tc.app, "PUT", "/api/v1/roles/"+adminRole.ID.String(), token, renamePayload)
	if err != nil {
		t.Fatalf("Failed to request rename Admin: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden when renaming Admin role, got %d. Body: %v", resp.StatusCode, parsed)
	}
}

func TestCheckPermissionLogic(t *testing.T) {
	tc := setupTestContext(t)

	// Admin permission check
	var adminRole auth.Role
	_ = tc.db.Where("name = ?", "Admin").First(&adminRole)
	hasAdminPerm, err := tc.service.CheckPermission(context.Background(), adminRole.ID, "any_module", "any_action")
	if err != nil {
		t.Fatalf("CheckPermission error: %v", err)
	}
	if !hasAdminPerm {
		t.Fatalf("Admin role should have all permissions")
	}

	// Custom role permission check
	customRole := role.Role{
		Name:        "TestCustomPerm_" + uuid.New().String()[:6],
		Color:       "#000000",
		Permissions: datatypes.JSON([]byte(`{"article": {"read": true, "create": false}}`)),
	}
	if err := tc.db.Create(&customRole).Error; err != nil {
		t.Fatalf("Failed to create custom role: %v", err)
	}
	defer tc.db.Unscoped().Delete(&customRole)

	hasRead, _ := tc.service.CheckPermission(context.Background(), customRole.ID, "article", "read")
	if !hasRead {
		t.Fatalf("Expected custom role to have article.read permission")
	}

	hasCreate, _ := tc.service.CheckPermission(context.Background(), customRole.ID, "article", "create")
	if hasCreate {
		t.Fatalf("Expected custom role to NOT have article.create permission")
	}

	hasDelete, _ := tc.service.CheckPermission(context.Background(), customRole.ID, "article", "delete")
	if hasDelete {
		t.Fatalf("Expected custom role to NOT have article.delete permission")
	}

	_ = strings.TrimSpace("")
}
