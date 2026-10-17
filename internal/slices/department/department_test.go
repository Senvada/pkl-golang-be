package department_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/infra/database"
	"pkl-golang-be/internal/middleware"
	"pkl-golang-be/internal/shared/response"
	"pkl-golang-be/internal/slices/auth"
	"pkl-golang-be/internal/slices/department"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type testDeptContext struct {
	app *fiber.App
	cfg *config.Config
	db  *gorm.DB
}

func setupDeptTest(t *testing.T) *testDeptContext {
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

	authSvc := auth.NewService(db, cfg)
	authHandler := auth.NewHandler(authSvc, cfg)
	auth.RegisterRoutes(api, authHandler, cfg)

	deptSvc := department.NewService(db, cfg)
	deptHandler := department.NewHandler(deptSvc, cfg)
	department.RegisterRoutes(api, deptHandler, cfg)

	return &testDeptContext{app: app, cfg: cfg, db: db}
}

func getAdminToken(t *testing.T, tc *testDeptContext) string {
	loginReq := auth.LoginRequest{
		Email:    tc.cfg.DefaultAdminEmail,
		Password: tc.cfg.DefaultAdminPassword,
	}
	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := tc.app.Test(req, 5000)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to login as admin: %v, status: %d", err, resp.StatusCode)
	}

	var jsonResp response.Envelope
	_ = json.NewDecoder(resp.Body).Decode(&jsonResp)
	dataMap, ok := jsonResp.Data.(map[string]any)
	if !ok {
		t.Fatalf("Invalid login response format")
	}
	token, ok := dataMap["access_token"].(string)
	if !ok || token == "" {
		token = dataMap["token"].(string)
	}
	return token
}

func TestDepartmentCRUD(t *testing.T) {
	tc := setupDeptTest(t)
	token := getAdminToken(t, tc)

	testCode := fmt.Sprintf("TEST%s", uuid.New().String()[:4])

	// 1. Create Department
	createPayload := department.CreateDepartmentRequest{
		Name:  "Test Jurusan Automated",
		Code:  testCode,
		Color: "#3B82F6",
	}
	body, _ := json.Marshal(createPayload)
	req := httptest.NewRequest("POST", "/api/v1/departments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := tc.app.Test(req, 5000)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create department: %v, status: %d", err, resp.StatusCode)
	}

	var createResp response.Envelope
	_ = json.NewDecoder(resp.Body).Decode(&createResp)
	deptData := createResp.Data.(map[string]any)
	createdID := deptData["id"].(string)

	// Clean up after test
	defer func() {
		_ = tc.db.Exec("DELETE FROM departments WHERE id = ?", createdID).Error
	}()

	// 2. Get Department by ID
	getReq := httptest.NewRequest("GET", "/api/v1/departments/"+createdID, nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getResp, err := tc.app.Test(getReq, 5000)
	if err != nil || getResp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to get department by ID: %v, status: %d", err, getResp.StatusCode)
	}

	// 3. Update Department
	newName := "Updated Test Jurusan"
	updatePayload := department.UpdateDepartmentRequest{
		Name: &newName,
	}
	upBody, _ := json.Marshal(updatePayload)
	upReq := httptest.NewRequest("PUT", "/api/v1/departments/"+createdID, bytes.NewReader(upBody))
	upReq.Header.Set("Content-Type", "application/json")
	upReq.Header.Set("Authorization", "Bearer "+token)

	upResp, err := tc.app.Test(upReq, 5000)
	if err != nil || upResp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to update department: %v, status: %d", err, upResp.StatusCode)
	}

	// 4. Delete Department
	delReq := httptest.NewRequest("DELETE", "/api/v1/departments/"+createdID, nil)
	delReq.Header.Set("Authorization", "Bearer "+token)
	delResp, err := tc.app.Test(delReq, 5000)
	if err != nil || delResp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to delete department: %v, status: %d", err, delResp.StatusCode)
	}
}
