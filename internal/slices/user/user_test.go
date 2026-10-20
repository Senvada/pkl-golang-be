package user_test

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
	"pkl-golang-be/internal/infra/storage"
	"pkl-golang-be/internal/middleware"
	"pkl-golang-be/internal/shared/response"
	"pkl-golang-be/internal/slices/auth"
	"pkl-golang-be/internal/slices/user"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type testContext struct {
	app      *fiber.App
	cfg      *config.Config
	db       *gorm.DB
	s3Client *storage.S3Client
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

	s3Client, err := storage.NewS3Client(cfg)
	if err != nil {
		t.Logf("Storage client not available for tests: %v", err)
	}

	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.CustomErrorHandler,
	})

	api := app.Group("/api/v1")

	// Wire auth
	authSvc := auth.NewService(db, cfg)
	authHandler := auth.NewHandler(authSvc, cfg)
	auth.RegisterRoutes(api, authHandler, cfg)

	// Wire user
	userSvc := user.NewService(db, cfg, s3Client)
	userHandler := user.NewHandler(userSvc, cfg)
	user.RegisterRoutes(api, userHandler, cfg)

	return &testContext{
		app:      app,
		cfg:      cfg,
		db:       db,
		s3Client: s3Client,
	}
}

func ensureRole(t *testing.T, db *gorm.DB) uuid.UUID {
	var role auth.Role
	err := db.Where("name = ?", "Siswa").First(&role).Error
	if err != nil {
		role = auth.Role{
			Name:        "Siswa",
			Color:       "#3B82F6",
			Permissions: []byte(`{"read":true}`),
		}
		if err := db.Create(&role).Error; err != nil {
			t.Fatalf("Failed to create role: %v", err)
		}
	}
	return role.ID
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

func TestUserProfileAndPresignFlow(t *testing.T) {
	tc := setupTestContext(t)
	roleID := ensureRole(t, tc.db)

	uniqueEmail := fmt.Sprintf("test_profile_%d@smkn6malang.sch.id", time.Now().UnixNano())
	initialPassword := "InitialPass123!"

	// 1. Register test user
	regResp, regBody, err := makeRequest(tc.app, "POST", "/api/v1/auth/register", "", map[string]any{
		"name":     "Profil Tester",
		"email":    uniqueEmail,
		"password": initialPassword,
		"role_id":  roleID.String(),
	})
	if err != nil || regResp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to register test user (status: %d): %v", regResp.StatusCode, regBody)
	}

	dataMap := regBody["data"].(map[string]any)
	token := dataMap["access_token"].(string)

	// Clean up user after test
	t.Cleanup(func() {
		tc.db.Exec("DELETE FROM tokens WHERE user_id IN (SELECT id FROM users WHERE email = ?)", uniqueEmail)
		tc.db.Exec("DELETE FROM users WHERE email = ?", uniqueEmail)
	})

	// 2. Get Me
	getResp, getBody, err := makeRequest(tc.app, "GET", "/api/v1/users/me", token, nil)
	if err != nil || getResp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to get profile: status %d, body: %v", getResp.StatusCode, getBody)
	}
	profileData := getBody["data"].(map[string]any)
	if profileData["name"] != "Profil Tester" {
		t.Errorf("Expected name 'Profil Tester', got '%v'", profileData["name"])
	}

	// 3. Update Name
	updatedName := "Nama Diperbarui"
	patchResp, patchBody, err := makeRequest(tc.app, "PATCH", "/api/v1/users/me", token, map[string]any{
		"name": updatedName,
	})
	if err != nil || patchResp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to patch name: status %d, body: %v", patchResp.StatusCode, patchBody)
	}
	patchedProfile := patchBody["data"].(map[string]any)
	if patchedProfile["name"] != updatedName {
		t.Errorf("Expected name '%s', got '%v'", updatedName, patchedProfile["name"])
	}

	// 4. Presign Avatar Upload URL (if S3 client is initialized)
	if tc.s3Client != nil {
		presignResp, presignBody, err := makeRequest(tc.app, "POST", "/api/v1/users/me/avatar/presign", token, map[string]any{
			"content_type": "image/webp",
			"file_size":    50000,
		})
		if err != nil || presignResp.StatusCode != http.StatusOK {
			t.Fatalf("Failed to presign avatar: status %d, body: %v", presignResp.StatusCode, presignBody)
		}
		presignData := presignBody["data"].(map[string]any)
		if presignData["upload_url"] == nil || presignData["public_url"] == nil {
			t.Errorf("Expected upload_url and public_url in presign response, got: %v", presignData)
		}

		publicURL := presignData["public_url"].(string)

		// 5. Update avatar_url in profile
		patchAvatarResp, patchAvatarBody, err := makeRequest(tc.app, "PATCH", "/api/v1/users/me", token, map[string]any{
			"avatar_url": publicURL,
		})
		if err != nil || patchAvatarResp.StatusCode != http.StatusOK {
			t.Fatalf("Failed to update avatar_url: status %d, body: %v", patchAvatarResp.StatusCode, patchAvatarBody)
		}
		avatarProfile := patchAvatarBody["data"].(map[string]any)
		if avatarProfile["avatar_url"] != publicURL {
			t.Errorf("Expected avatar_url '%s', got '%v'", publicURL, avatarProfile["avatar_url"])
		}

		// 6. Delete Avatar
		delResp, delBody, err := makeRequest(tc.app, "DELETE", "/api/v1/users/me/avatar", token, nil)
		if err != nil || delResp.StatusCode != http.StatusOK {
			t.Fatalf("Failed to delete avatar: status %d, body: %v", delResp.StatusCode, delBody)
		}
		delProfile := delBody["data"].(map[string]any)
		if delProfile["avatar_url"] != nil {
			t.Errorf("Expected avatar_url to be nil after deletion, got: %v", delProfile["avatar_url"])
		}
	}

	// 7. Change Password - Wrong old password
	wrongPassResp, _, err := makeRequest(tc.app, "PUT", "/api/v1/users/me/password", token, map[string]any{
		"old_password": "WrongPassword!",
		"new_password": "NewSecretPass123!",
	})
	if err != nil || wrongPassResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400 for wrong old password, got %d", wrongPassResp.StatusCode)
	}

	// 8. Change Password - Correct old password
	newPass := "NewSecretPass123!"
	correctPassResp, _, err := makeRequest(tc.app, "PUT", "/api/v1/users/me/password", token, map[string]any{
		"old_password": initialPassword,
		"new_password": newPass,
	})
	if err != nil || correctPassResp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to change password: got status %d", correctPassResp.StatusCode)
	}

	// 9. Login with new password to verify
	loginResp, loginBody, err := makeRequest(tc.app, "POST", "/api/v1/auth/login", "", map[string]any{
		"email":    uniqueEmail,
		"password": newPass,
	})
	if err != nil || loginResp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to login with new password: status %d, body: %v", loginResp.StatusCode, loginBody)
	}
	_ = response.Single
}
