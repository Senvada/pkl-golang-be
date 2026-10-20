package company_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/infra/database"
	"pkl-golang-be/internal/infra/seeder"
	"pkl-golang-be/internal/infra/storage"
	"pkl-golang-be/internal/middleware"
	"pkl-golang-be/internal/shared/response"
	"pkl-golang-be/internal/slices/auth"
	"pkl-golang-be/internal/slices/company"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type testCompanyContext struct {
	app      *fiber.App
	cfg      *config.Config
	db       *gorm.DB
	s3Client *storage.S3Client
}

func setupCompanyTest(t *testing.T) *testCompanyContext {
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	s3Client, err := storage.NewS3Client(cfg)
	if err != nil {
		t.Logf("Storage client not available for tests: %v", err)
	}

	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.CustomErrorHandler,
	})

	api := app.Group("/api/v1")

	authSvc := auth.NewService(db, cfg)
	authHandler := auth.NewHandler(authSvc, cfg)
	auth.RegisterRoutes(api, authHandler, cfg)

	compSvc := company.NewService(db, cfg, s3Client)
	compHandler := company.NewHandler(compSvc, cfg)
	company.RegisterRoutes(api, compHandler, cfg)

	return &testCompanyContext{
		app:      app,
		cfg:      cfg,
		db:       db,
		s3Client: s3Client,
	}
}

func getAdminToken(t *testing.T, tc *testCompanyContext) string {
	_ = seeder.SeedDefaultAdmin(tc.db, tc.cfg, seeder.Options{ForceUpdatePassword: true})

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

func TestCompanyLifecycleAndDepartments(t *testing.T) {
	tc := setupCompanyTest(t)
	token := getAdminToken(t, tc)

	// Fetch a department (e.g. RPL)
	var deptIDStr string
	_ = tc.db.Table("departments").Where("code = ?", "RPL").Select("id::text").Scan(&deptIDStr).Error
	deptID, _ := uuid.Parse(deptIDStr)

	uniqueName := fmt.Sprintf("PT Teknologi Solusi %d", time.Now().UnixNano())
	address := "Jl. Industri Digital Megah No. 10, Malang"
	phone := "08123456789"
	lat := -7.9825
	lng := 112.6308

	// 1. Create Company with accepted departments
	createPayload := company.CreateCompanyRequest{
		Name:          uniqueName,
		Address:       &address,
		PhoneNumber:   &phone,
		Lat:           &lat,
		Lng:           &lng,
		Status:        "active",
		DepartmentIDs: []uuid.UUID{deptID},
		Content:       nil,
	}
	body, _ := json.Marshal(createPayload)
	req := httptest.NewRequest("POST", "/api/v1/companies", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := tc.app.Test(req, 5000)
	if err != nil {
		t.Fatalf("Test request failed: %v", err)
	}
	respBodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create company (status %d): %s", resp.StatusCode, string(respBodyBytes))
	}

	var createResp response.Envelope
	_ = json.Unmarshal(respBodyBytes, &createResp)
	compData := createResp.Data.(map[string]any)
	createdID := compData["id"].(string)
	slug := compData["slug"].(string)

	defer func() {
		_ = tc.db.Exec("DELETE FROM companies_departments WHERE company_id = ?", createdID).Error
		_ = tc.db.Exec("DELETE FROM companies WHERE id = ?", createdID).Error
	}()

	// Verify departments list
	depts := compData["departments"].([]any)
	if len(depts) == 0 {
		t.Fatalf("Expected company to have mapped departments, got 0")
	}

	// 2. Get Public by Slug
	slugReq := httptest.NewRequest("GET", "/api/v1/companies/slug/"+slug, nil)
	slugResp, err := tc.app.Test(slugReq, 5000)
	if err != nil || slugResp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to get company by slug: %v, status: %d", err, slugResp.StatusCode)
	}

	// 3. Get Maps endpoint (verifying coordinates exist)
	mapReq := httptest.NewRequest("GET", "/api/v1/companies/maps", nil)
	mapResp, err := tc.app.Test(mapReq, 5000)
	if err != nil || mapResp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to get company maps: %v, status: %d", err, mapResp.StatusCode)
	}

	// 4. Update Company Details
	updatedName := uniqueName + " Sukses"
	updatePayload := company.UpdateCompanyRequest{
		Name: &updatedName,
	}
	upBody, _ := json.Marshal(updatePayload)
	upReq := httptest.NewRequest("PUT", "/api/v1/companies/"+createdID, bytes.NewReader(upBody))
	upReq.Header.Set("Content-Type", "application/json")
	upReq.Header.Set("Authorization", "Bearer "+token)

	upResp, err := tc.app.Test(upReq, 5000)
	if err != nil || upResp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to update company: %v, status: %d", err, upResp.StatusCode)
	}

	// 5. Update Rich Content (PATCH)
	contentJSON := datatypes.JSON([]byte(`{"type":"doc","content":[{"type":"paragraph","text":"Profil perusahaan mitra resmi"}]}`))
	patchPayload := company.UpdateCompanyContentRequest{
		Content: contentJSON,
	}
	patchBody, _ := json.Marshal(patchPayload)
	patchReq := httptest.NewRequest("PATCH", "/api/v1/companies/"+createdID+"/content", bytes.NewReader(patchBody))
	patchReq.Header.Set("Content-Type", "application/json")
	patchReq.Header.Set("Authorization", "Bearer "+token)

	patchResp, err := tc.app.Test(patchReq, 5000)
	if err != nil || patchResp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to patch company content: %v, status: %d", err, patchResp.StatusCode)
	}

	// 6. Resolve Maps Location
	resolvePayload := company.ResolveMapsRequest{
		URL: "https://www.google.com/maps/place/SMK+Negeri+6+Malang/@-7.9825678,112.6308123,17z/data=!3m1!4b1",
	}
	resBody, _ := json.Marshal(resolvePayload)
	resReq := httptest.NewRequest("POST", "/api/v1/companies/maps/resolve", bytes.NewReader(resBody))
	resReq.Header.Set("Content-Type", "application/json")
	resReq.Header.Set("Authorization", "Bearer "+token)

	resResp, err := tc.app.Test(resReq, 5000)
	if err != nil || resResp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to resolve maps location: %v, status: %d", err, resResp.StatusCode)
	}

	var resData response.Envelope
	resBodyBytes, _ := io.ReadAll(resResp.Body)
	_ = json.Unmarshal(resBodyBytes, &resData)
	coords := resData.Data.(map[string]any)
	if coords["lat"].(float64) != -7.9825678 || coords["lng"].(float64) != 112.6308123 {
		t.Fatalf("Unexpected resolved coordinates: %v", coords)
	}

	// 7. Delete Company
	delReq := httptest.NewRequest("DELETE", "/api/v1/companies/"+createdID, nil)
	delReq.Header.Set("Authorization", "Bearer "+token)
	delResp, err := tc.app.Test(delReq, 5000)
	if err != nil || delResp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to delete company: %v, status: %d", err, delResp.StatusCode)
	}
}

func TestCompanyImagePresign(t *testing.T) {
	tc := setupCompanyTest(t)
	if tc.s3Client == nil {
		t.Skip("S3 client not configured, skipping presign test")
	}
	token := getAdminToken(t, tc)

	// 1. Test POST /api/v1/companies/image/presign (no ID - for creating new company)
	presignPayload := company.PresignImageRequest{
		ContentType: "image/jpeg",
		FileSize:    385550,
	}
	body, _ := json.Marshal(presignPayload)
	req := httptest.NewRequest("POST", "/api/v1/companies/image/presign", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := tc.app.Test(req, 5000)
	if err != nil || resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Failed to get presigned URL without company ID: %v, status: %d, body: %s", err, resp.StatusCode, string(respBody))
	}

	var presignResp response.Envelope
	_ = json.NewDecoder(resp.Body).Decode(&presignResp)
	pData := presignResp.Data.(map[string]any)
	if pData["upload_url"] == nil || pData["public_url"] == nil {
		t.Fatalf("Expected upload_url and public_url in response, got: %v", pData)
	}

	// 2. Test POST /api/v1/companies/:id/image/presign (with company ID)
	dummyID := uuid.New().String()
	reqWithID := httptest.NewRequest("POST", "/api/v1/companies/"+dummyID+"/image/presign", bytes.NewReader(body))
	reqWithID.Header.Set("Content-Type", "application/json")
	reqWithID.Header.Set("Authorization", "Bearer "+token)

	respWithID, err := tc.app.Test(reqWithID, 5000)
	if err != nil || respWithID.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(respWithID.Body)
		t.Fatalf("Failed to get presigned URL with company ID: %v, status: %d, body: %s", err, respWithID.StatusCode, string(respBody))
	}
}
