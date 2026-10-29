package assessment_test

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
	"pkl-golang-be/internal/slices/assessment"
	"pkl-golang-be/internal/slices/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

type testContext struct {
	app *fiber.App
	cfg *config.Config
	db  *gorm.DB
}

func setupTest(t *testing.T) *testContext {
	_ = godotenv.Load("../../../.env")
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

	assessmentSvc := assessment.NewService(db, cfg)
	assessmentHandler := assessment.NewHandler(assessmentSvc, cfg)
	assessment.RegisterRoutes(api, assessmentHandler, cfg)

	return &testContext{app: app, cfg: cfg, db: db}
}

func getAdminToken(t *testing.T, tc *testContext) string {
	loginReq := auth.LoginRequest{
		Email:    tc.cfg.DefaultAdminEmail,
		Password: tc.cfg.DefaultAdminPassword,
	}
	body, _ := json.Marshal(loginReq)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := tc.app.Test(req, 10000)
	if err != nil {
		t.Fatalf("Login request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Admin login returned status: %d", resp.StatusCode)
	}

	var res response.Envelope
	_ = json.NewDecoder(resp.Body).Decode(&res)

	dataMap, ok := res.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Unexpected login response payload")
	}

	return dataMap["access_token"].(string)
}

func TestAcademicPeriods(t *testing.T) {
	tc := setupTest(t)
	token := getAdminToken(t, tc)

	// 1. Get Academic Periods
	req := httptest.NewRequest(http.MethodGet, "/api/v1/academic-periods", nil)
	resp, err := tc.app.Test(req)
	if err != nil {
		t.Fatalf("Get periods failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	// 2. Get Active Academic Period
	reqActive := httptest.NewRequest(http.MethodGet, "/api/v1/academic-periods/active", nil)
	respActive, err := tc.app.Test(reqActive)
	if err != nil {
		t.Fatalf("Get active period failed: %v", err)
	}
	if respActive.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", respActive.StatusCode)
	}

	// 3. Create New Academic Period
	newPeriodName := fmt.Sprintf("2026/2027-%s", uuid.NewString()[:6])
	createReq := assessment.CreateAcademicPeriodRequest{
		Name:     newPeriodName,
		IsActive: false,
	}
	body, _ := json.Marshal(createReq)
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/academic-periods", bytes.NewReader(body))
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.Header.Set("Authorization", "Bearer "+token)

	respCreate, err := tc.app.Test(reqCreate)
	if err != nil {
		t.Fatalf("Create period failed: %v", err)
	}
	if respCreate.StatusCode != http.StatusCreated {
		t.Errorf("Expected 201, got %d", respCreate.StatusCode)
	}
}

func TestAssessmentTemplates(t *testing.T) {
	tc := setupTest(t)
	token := getAdminToken(t, tc)

	// 1. Get Assessment Templates
	req := httptest.NewRequest(http.MethodGet, "/api/v1/assessment-templates", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := tc.app.Test(req)
	if err != nil {
		t.Fatalf("Get templates failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	var res response.Envelope
	_ = json.NewDecoder(resp.Body).Decode(&res)
	items, ok := res.Data.([]interface{})
	if !ok || len(items) == 0 {
		t.Fatalf("Expected seeded templates, got %v", res.Data)
	}

	firstTpl := items[0].(map[string]interface{})
	tplID := firstTpl["id"].(string)

	// 2. Get Template By ID
	reqByID := httptest.NewRequest(http.MethodGet, "/api/v1/assessment-templates/"+tplID, nil)
	reqByID.Header.Set("Authorization", "Bearer "+token)
	respByID, err := tc.app.Test(reqByID)
	if err != nil {
		t.Fatalf("Get template by ID failed: %v", err)
	}
	if respByID.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", respByID.StatusCode)
	}

	// 3. Update Template Title
	updateReq := assessment.UpdateAssessmentTemplateRequest{
		Title: "Updated Template Title Test",
		WorkAspects: []assessment.AspectTemplateItemDTO{
			{ID: uuid.NewString(), Category: "Web", Title: "Custom Technical Indicator"},
		},
		BehaviorAspects: []assessment.AspectTemplateItemDTO{
			{ID: uuid.NewString(), Title: "Disiplin"},
		},
	}
	upBody, _ := json.Marshal(updateReq)
	reqUpdate := httptest.NewRequest(http.MethodPut, "/api/v1/assessment-templates/"+tplID, bytes.NewReader(upBody))
	reqUpdate.Header.Set("Content-Type", "application/json")
	reqUpdate.Header.Set("Authorization", "Bearer "+token)

	respUpdate, err := tc.app.Test(reqUpdate)
	if err != nil {
		t.Fatalf("Update template failed: %v", err)
	}
	if respUpdate.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", respUpdate.StatusCode)
	}
}

func TestGradeCalculation(t *testing.T) {
	// A = 86 - 100 = Sangat Baik
	// B = 71 - 85  = Baik
	// C = 56 - 70  = Cukup
	// D = 0 - 55   = Kurang
	letter, pred := assessment.CalculateGrade(90)
	if letter != "A" || pred != "Sangat Baik" {
		t.Errorf("Expected A, Sangat Baik; got %s, %s", letter, pred)
	}

	letter, pred = assessment.CalculateGrade(80)
	if letter != "B" || pred != "Baik" {
		t.Errorf("Expected B, Baik; got %s, %s", letter, pred)
	}

	letter, pred = assessment.CalculateGrade(65)
	if letter != "C" || pred != "Cukup" {
		t.Errorf("Expected C, Cukup; got %s, %s", letter, pred)
	}

	letter, pred = assessment.CalculateGrade(50)
	if letter != "D" || pred != "Kurang" {
		t.Errorf("Expected D, Kurang; got %s, %s", letter, pred)
	}
}

func TestStudentAssessmentFlow(t *testing.T) {
	tc := setupTest(t)

	// Create test student user
	var rplDept assessment.Department
	if err := tc.db.Where("code = ?", "RPL").First(&rplDept).Error; err != nil {
		t.Fatalf("RPL department not found: %v", err)
	}

	var siswaRole auth.Role
	if err := tc.db.Where("name = ?", "Siswa").First(&siswaRole).Error; err != nil {
		t.Fatalf("Siswa role not found: %v", err)
	}

	studentID := uuid.New()
	testStudent := auth.User{
		ID:           studentID,
		RoleID:       siswaRole.ID,
		DepartmentID: &rplDept.ID,
		Name:         "Budi Santoso",
		Email:        fmt.Sprintf("budi-%s@test.com", studentID.String()[:8]),
		Password:     "password123",
		Status:       "active",
	}
	if err := tc.db.Create(&testStudent).Error; err != nil {
		t.Fatalf("Failed to create test student: %v", err)
	}

	// Generate JWT for student
	jwtToken, _, err := auth.GenerateAccessToken(&testStudent, "Siswa", tc.cfg.JWTSecret)
	if err != nil {
		t.Fatalf("Failed to generate student token: %v", err)
	}

	// 1. Get My Assessment (draft prefilled from template)
	reqDraft := httptest.NewRequest(http.MethodGet, "/api/v1/student-assessments/my", nil)
	reqDraft.Header.Set("Authorization", "Bearer "+jwtToken)
	respDraft, err := tc.app.Test(reqDraft)
	if err != nil {
		t.Fatalf("Get my assessment draft failed: %v", err)
	}
	if respDraft.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", respDraft.StatusCode)
	}

	var resDraft response.Envelope
	_ = json.NewDecoder(respDraft.Body).Decode(&resDraft)
	dataMap := resDraft.Data.(map[string]interface{})

	workAspects := dataMap["work_aspects"].([]interface{})
	if len(workAspects) == 0 {
		t.Fatalf("Expected prefilled work aspects from RPL template, got 0")
	}

	// 2. Student fills scores and adds a custom indicator!
	workItems := []assessment.WorkAspectItemDTO{}
	for _, item := range workAspects {
		iMap := item.(map[string]interface{})
		workItems = append(workItems, assessment.WorkAspectItemDTO{
			ID:           iMap["id"].(string),
			Category:     iMap["category"].(string),
			Title:        iMap["title"].(string),
			IsCustom:     false,
			ScoreNumeric: 85, // B
		})
	}
	// Add custom item
	workItems = append(workItems, assessment.WorkAspectItemDTO{
		ID:           uuid.NewString(),
		Category:     "Specialized",
		Title:        "Pengembangan Modul Machine Learning TensorFlow",
		IsCustom:     true,
		ScoreNumeric: 95, // A
	})

	behaviorAspects := dataMap["behavior_aspects"].([]interface{})
	behItems := []assessment.BehaviorAspectItemDTO{}
	for _, item := range behaviorAspects {
		iMap := item.(map[string]interface{})
		behItems = append(behItems, assessment.BehaviorAspectItemDTO{
			ID:           iMap["id"].(string),
			Title:        iMap["title"].(string),
			ScoreNumeric: 88, // A
		})
	}

	dateStr := "2026-10-25"
	upsertReq := assessment.UpsertStudentAssessmentRequest{
		StudentNIS:      "12345/001.065",
		StudentClass:    "XII RPL 1",
		CompanyName:     "PT Inovasi Digital Nusantara",
		CompanyAddress:  "Jl. Soekarno Hatta No. 9, Malang",
		SupervisorName:  "Ir. Bambang Wijaya",
		SupervisorNIP:   "198501012010011002",
		AssessmentCity:  "Malang",
		AssessmentDate:  &dateStr,
		WorkAspects:     workItems,
		BehaviorAspects: behItems,
		Status:          "submitted",
	}

	body, _ := json.Marshal(upsertReq)
	reqSave := httptest.NewRequest(http.MethodPost, "/api/v1/student-assessments", bytes.NewReader(body))
	reqSave.Header.Set("Content-Type", "application/json")
	reqSave.Header.Set("Authorization", "Bearer "+jwtToken)

	respSave, err := tc.app.Test(reqSave)
	if err != nil {
		t.Fatalf("Submit assessment failed: %v", err)
	}
	if respSave.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", respSave.StatusCode)
	}

	var resSaved response.Envelope
	_ = json.NewDecoder(respSave.Body).Decode(&resSaved)
	savedMap := resSaved.Data.(map[string]interface{})

	if savedMap["status"] != "submitted" {
		t.Errorf("Expected status submitted, got %v", savedMap["status"])
	}
	if savedMap["work_average"].(float64) <= 0 {
		t.Errorf("Expected work_average > 0, got %v", savedMap["work_average"])
	}
	if savedMap["overall_average"].(float64) <= 0 {
		t.Errorf("Expected overall_average > 0, got %v", savedMap["overall_average"])
	}
}
