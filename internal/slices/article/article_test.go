package article_test

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
	"pkl-golang-be/internal/slices/article"
	"pkl-golang-be/internal/slices/auth"
	"pkl-golang-be/internal/slices/company"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type testArticleContext struct {
	app      *fiber.App
	cfg      *config.Config
	db       *gorm.DB
	s3Client *storage.S3Client
}

func setupArticleTest(t *testing.T) *testArticleContext {
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

	artSvc := article.NewService(db, cfg, s3Client)
	artHandler := article.NewHandler(artSvc, cfg)
	article.RegisterRoutes(api, artHandler, cfg)

	return &testArticleContext{
		app:      app,
		cfg:      cfg,
		db:       db,
		s3Client: s3Client,
	}
}

func getAdminToken(t *testing.T, tc *testArticleContext) string {
	_ = seeder.EnsureDefaultAdmin(tc.db, tc.cfg)

	loginPayload := auth.LoginRequest{
		Email:    tc.cfg.DefaultAdminEmail,
		Password: tc.cfg.DefaultAdminPassword,
	}
	body, _ := json.Marshal(loginPayload)

	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := tc.app.Test(req, 10000)
	if err != nil {
		t.Fatalf("Login request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Admin login returned non-200 (%d): %s", resp.StatusCode, string(respBody))
	}

	var jsonResp response.Envelope
	_ = json.NewDecoder(resp.Body).Decode(&jsonResp)

	dataMap, ok := jsonResp.Data.(map[string]any)
	if !ok {
		t.Fatalf("Failed to parse login response data: %+v", jsonResp.Data)
	}

	token, ok := dataMap["access_token"].(string)
	if !ok || token == "" {
		token, _ = dataMap["token"].(string)
	}
	if token == "" {
		t.Fatalf("Token not found in login response: %+v", dataMap)
	}

	return token
}

func createTestCompany(t *testing.T, tc *testArticleContext, token string) string {
	uniqueName := fmt.Sprintf("Mitra Artikel %d", time.Now().UnixNano())
	createPayload := company.CreateCompanyRequest{
		Name:   uniqueName,
		Status: "active",
	}
	body, _ := json.Marshal(createPayload)

	req := httptest.NewRequest("POST", "/api/v1/companies", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := tc.app.Test(req, 10000)
	if err != nil {
		t.Fatalf("Create company failed: %v", err)
	}
	defer resp.Body.Close()

	var jsonResp response.Envelope
	_ = json.NewDecoder(resp.Body).Decode(&jsonResp)

	dataMap := jsonResp.Data.(map[string]any)
	return dataMap["id"].(string)
}

func TestArticleLifecycleWithEmptyContent(t *testing.T) {
	tc := setupArticleTest(t)
	token := getAdminToken(t, tc)
	companyIDStr := createTestCompany(t, tc, token)
	companyID, _ := uuid.Parse(companyIDStr)

	uniqueTitle := fmt.Sprintf("Panduan Magang Industri %d", time.Now().UnixNano())
	coverURL := "http://127.0.0.1:54326/object/public/pkl-assets/articles/staging/initial_cover.webp"
	styleJSON := datatypes.JSON([]byte(`{"type": 2, "height": 450}`))

	// 1. Create Article with empty content
	createPayload := article.CreateArticleRequest{
		Title:     uniqueTitle,
		CompanyID: companyID,
		CoverURL:  &coverURL,
		Style:     &styleJSON,
		Status:    "draft",
	}
	body, _ := json.Marshal(createPayload)

	req := httptest.NewRequest("POST", "/api/v1/articles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := tc.app.Test(req, 10000)
	if err != nil {
		t.Fatalf("Create article failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 201 Created, got %d: %s", resp.StatusCode, string(respBody))
	}

	var createResp response.Envelope
	_ = json.NewDecoder(resp.Body).Decode(&createResp)
	createdData := createResp.Data.(map[string]any)
	createdID := createdData["id"].(string)
	createdSlug := createdData["slug"].(string)

	if createdID == "" {
		t.Fatal("Article ID is empty")
	}

	// Verify content is empty array "[]"
	rawContent, _ := json.Marshal(createdData["content"])
	if string(rawContent) != "[]" {
		t.Fatalf("Expected empty array '[]' for initial content, got %s", string(rawContent))
	}

	// 2. Fetch Article by ID
	getReq := httptest.NewRequest("GET", "/api/v1/articles/"+createdID, nil)
	getResp, err := tc.app.Test(getReq, 10000)
	if err != nil {
		t.Fatalf("Get article by ID failed: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", getResp.StatusCode)
	}

	// 3. Fetch Article by Slug
	slugReq := httptest.NewRequest("GET", "/api/v1/articles/slug/"+createdSlug, nil)
	slugResp, err := tc.app.Test(slugReq, 10000)
	if err != nil {
		t.Fatalf("Get article by slug failed: %v", err)
	}
	defer slugResp.Body.Close()
	if slugResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", slugResp.StatusCode)
	}

	// 4. Presign Cover URL
	presignPayload := article.PresignCoverRequest{
		ContentType: "image/webp",
		FileSize:    256000,
	}
	preBody, _ := json.Marshal(presignPayload)
	preReq := httptest.NewRequest("POST", "/api/v1/articles/"+createdID+"/cover/presign", bytes.NewReader(preBody))
	preReq.Header.Set("Content-Type", "application/json")
	preReq.Header.Set("Authorization", "Bearer "+token)

	preResp, err := tc.app.Test(preReq, 10000)
	if err != nil {
		t.Fatalf("Presign request failed: %v", err)
	}
	defer preResp.Body.Close()
	if preResp.StatusCode != http.StatusOK {
		preRespBody, _ := io.ReadAll(preResp.Body)
		t.Fatalf("Presign returned non-200 (%d): %s", preResp.StatusCode, string(preRespBody))
	}

	// 5. Update Article (Cover and publish)
	updatedTitle := uniqueTitle + " (Updated)"
	newCover := "http://127.0.0.1:54326/object/public/pkl-assets/articles/" + createdID + "/updated_cover.webp"
	publishedStatus := "published"
	updatePayload := article.UpdateArticleRequest{
		Title:    &updatedTitle,
		CoverURL: &newCover,
		Status:   &publishedStatus,
	}
	upBody, _ := json.Marshal(updatePayload)
	upReq := httptest.NewRequest("PUT", "/api/v1/articles/"+createdID, bytes.NewReader(upBody))
	upReq.Header.Set("Content-Type", "application/json")
	upReq.Header.Set("Authorization", "Bearer "+token)

	upResp, err := tc.app.Test(upReq, 10000)
	if err != nil {
		t.Fatalf("Update article failed: %v", err)
	}
	defer upResp.Body.Close()
	if upResp.StatusCode != http.StatusOK {
		upRespBody, _ := io.ReadAll(upResp.Body)
		t.Fatalf("Update returned non-200 (%d): %s", upResp.StatusCode, string(upRespBody))
	}

	// 6. Delete Article
	delReq := httptest.NewRequest("DELETE", "/api/v1/articles/"+createdID, nil)
	delReq.Header.Set("Authorization", "Bearer "+token)

	delResp, err := tc.app.Test(delReq, 10000)
	if err != nil {
		t.Fatalf("Delete article failed: %v", err)
	}
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusOK {
		t.Fatalf("Delete returned non-200 (%d)", delResp.StatusCode)
	}
}
