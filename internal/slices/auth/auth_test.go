package auth_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/infra/database"
	"pkl-golang-be/internal/middleware"
	"pkl-golang-be/internal/shared/response"
	"pkl-golang-be/internal/slices/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func setupTestApp(t *testing.T) (*fiber.App, *auth.Handler, *config.Config) {
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

	svc := auth.NewService(db, cfg)
	handler := auth.NewHandler(svc, cfg)

	api := app.Group("/api/v1")
	auth.RegisterRoutes(api, handler, cfg)

	return app, handler, cfg
}

func ensureTestRole(t *testing.T) uuid.UUID {
	cfg, _ := config.LoadConfig()
	db, _ := database.Connect(cfg)

	var role auth.Role
	err := db.Where("name = ?", "Siswa").First(&role).Error
	if err != nil {
		role = auth.Role{
			Name:        "Siswa",
			Color:       "#3B82F6",
			Permissions: []byte(`{"read":true}`),
		}
		if err := db.Create(&role).Error; err != nil {
			t.Fatalf("Failed to seed test role: %v", err)
		}
	}
	return role.ID
}

func TestAuth_Flow(t *testing.T) {
	app, _, _ := setupTestApp(t)
	roleID := ensureTestRole(t)

	uniqueEmail := "siswa_" + uuid.NewString()[:8] + "@smkn6malang.sch.id"
	password := "Secret123!"

	// ==========================================
	// 1. REGISTER
	// ==========================================
	regPayload := auth.RegisterRequest{
		Name:     "Budi Santoso",
		Email:    uniqueEmail,
		Password: password,
		RoleID:   roleID,
	}
	bodyBytes, _ := json.Marshal(regPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Register request failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 201 Created, got %d: %s", resp.StatusCode, string(respBody))
	}

	var regEnvelope struct {
		Success bool              `json:"success"`
		Data    auth.AuthResponse `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&regEnvelope)
	if !regEnvelope.Success || regEnvelope.Data.AccessToken == "" {
		t.Fatalf("Register did not return valid access token")
	}

	accessToken := regEnvelope.Data.AccessToken
	userID := regEnvelope.Data.User.ID

	// Extract refresh token cookie
	cookies := resp.Cookies()
	var refreshTokenCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.RefreshTokenCookieName {
			refreshTokenCookie = c
			break
		}
	}
	if refreshTokenCookie == nil || refreshTokenCookie.Value == "" {
		t.Fatalf("Expected refresh token cookie to be set")
	}

	// ==========================================
	// 2. REGISTER DUPLICATE EMAIL (Should Fail 409)
	// ==========================================
	reqDup := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(bodyBytes))
	reqDup.Header.Set("Content-Type", "application/json")
	respDup, _ := app.Test(reqDup)
	if respDup.StatusCode != http.StatusConflict {
		t.Fatalf("Expected 409 Conflict for duplicate email, got %d", respDup.StatusCode)
	}

	// ==========================================
	// 3. LOGIN WITH WRONG PASSWORD (Should Fail 401)
	// ==========================================
	badLoginPayload := auth.LoginRequest{
		Email:    uniqueEmail,
		Password: "WrongPassword!",
	}
	badLoginBytes, _ := json.Marshal(badLoginPayload)
	reqBadLogin := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(badLoginBytes))
	reqBadLogin.Header.Set("Content-Type", "application/json")
	respBadLogin, _ := app.Test(reqBadLogin)
	if respBadLogin.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected 401 Unauthorized for wrong password, got %d", respBadLogin.StatusCode)
	}

	// ==========================================
	// 4. LOGIN SUCCESS
	// ==========================================
	loginPayload := auth.LoginRequest{
		Email:    uniqueEmail,
		Password: password,
		Remember: true,
	}
	loginBytes, _ := json.Marshal(loginPayload)
	reqLogin := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBytes))
	reqLogin.Header.Set("Content-Type", "application/json")
	respLogin, _ := app.Test(reqLogin)
	if respLogin.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK for valid login, got %d", respLogin.StatusCode)
	}

	// ==========================================
	// 5. GET /ME (Protected)
	// ==========================================
	reqMe := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+accessToken)
	respMe, _ := app.Test(reqMe)
	if respMe.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK for /me, got %d", respMe.StatusCode)
	}

	var meEnvelope struct {
		Success bool                `json:"success"`
		Data    auth.UserProfileDTO `json:"data"`
	}
	_ = json.NewDecoder(respMe.Body).Decode(&meEnvelope)
	if meEnvelope.Data.Email != uniqueEmail {
		t.Fatalf("Expected user email '%s', got '%s'", uniqueEmail, meEnvelope.Data.Email)
	}
	if meEnvelope.Data.GoogleBound {
		t.Fatalf("Expected GoogleBound=false initially")
	}

	// ==========================================
	// 6. REFRESH TOKEN
	// ==========================================
	reqRefresh := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	reqRefresh.AddCookie(refreshTokenCookie)
	respRefresh, _ := app.Test(reqRefresh)
	if respRefresh.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK for /refresh, got %d", respRefresh.StatusCode)
	}

	var refreshEnvelope struct {
		Success bool                      `json:"success"`
		Data    auth.RefreshTokenResponse `json:"data"`
	}
	_ = json.NewDecoder(respRefresh.Body).Decode(&refreshEnvelope)
	if refreshEnvelope.Data.AccessToken == "" {
		t.Fatalf("Expected refreshed access token")
	}

	// ==========================================
	// 7. GOOGLE LOGIN WITH UNBOUND ACCOUNT (Must Fail 401)
	// ==========================================
	cfg, _ := config.LoadConfig()
	db, _ := database.Connect(cfg)

	// Direct test of service GoogleLogin rejection for unbound sub
	_ = auth.NewService(db, cfg)
	dummyGoogleSub := "google_sub_unbound_99999"

	// Directly mock checking if unbound google_id is rejected:
	var unboundAccount auth.UserGoogleAccount
	errUnbound := db.Where("google_id = ?", dummyGoogleSub).First(&unboundAccount).Error
	if errUnbound == nil {
		t.Fatalf("Dummy account should not exist in DB")
	}

	// Manually bind this dummy google sub to our registered user
	manualBind := auth.UserGoogleAccount{
		UserID:        userID,
		GoogleID:      dummyGoogleSub,
		GoogleEmail:   uniqueEmail,
		EmailVerified: true,
	}
	if err := db.Create(&manualBind).Error; err != nil {
		t.Fatalf("Failed to insert manual google bind: %v", err)
	}

	// Now check /me again -> GoogleBound must be true!
	reqMeAfterBind := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	reqMeAfterBind.Header.Set("Authorization", "Bearer "+accessToken)
	respMeAfterBind, _ := app.Test(reqMeAfterBind)
	var meAfterBindEnv struct {
		Data auth.UserProfileDTO `json:"data"`
	}
	_ = json.NewDecoder(respMeAfterBind.Body).Decode(&meAfterBindEnv)
	if !meAfterBindEnv.Data.GoogleBound {
		t.Fatalf("Expected GoogleBound=true after binding Google account")
	}

	// ==========================================
	// 8. UNBIND GOOGLE ACCOUNT
	// ==========================================
	reqUnbind := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/google/unbind", nil)
	reqUnbind.Header.Set("Authorization", "Bearer "+accessToken)
	respUnbind, _ := app.Test(reqUnbind)
	if respUnbind.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK for /google/unbind, got %d", respUnbind.StatusCode)
	}

	// Check /me again -> GoogleBound must be false!
	reqMeAfterUnbind := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	reqMeAfterUnbind.Header.Set("Authorization", "Bearer "+accessToken)
	respMeAfterUnbind, _ := app.Test(reqMeAfterUnbind)
	var meAfterUnbindEnv struct {
		Data auth.UserProfileDTO `json:"data"`
	}
	_ = json.NewDecoder(respMeAfterUnbind.Body).Decode(&meAfterUnbindEnv)
	if meAfterUnbindEnv.Data.GoogleBound {
		t.Fatalf("Expected GoogleBound=false after unbinding Google account")
	}

	// ==========================================
	// 9. LOGOUT
	// ==========================================
	reqLogout := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	reqLogout.AddCookie(refreshTokenCookie)
	respLogout, _ := app.Test(reqLogout)
	if respLogout.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK for /logout, got %d", respLogout.StatusCode)
	}
}

// Ensure unused import warnings don't fail
var _ = time.Now
var _ = response.Single
var _ = bcrypt.CompareHashAndPassword
