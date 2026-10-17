package user_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"pkl-golang-be/internal/infra/seeder"
	"pkl-golang-be/internal/slices/user"

	"github.com/google/uuid"
)

func TestUserManagement_HierarchyAndScoping(t *testing.T) {
	tc := setupTestContext(t)

	// Ensure admin password matches config
	_ = seeder.SeedDefaultAdmin(tc.db, tc.cfg, seeder.Options{ForceUpdatePassword: true})

	// Login as default admin
	adminLoginResp, adminLoginBody, err := makeRequest(tc.app, "POST", "/api/v1/auth/login", "", map[string]any{
		"email":    tc.cfg.DefaultAdminEmail,
		"password": tc.cfg.DefaultAdminPassword,
	})
	if err != nil || adminLoginResp.StatusCode != http.StatusOK {
		t.Fatalf("Admin login failed (status %d): %v", adminLoginResp.StatusCode, adminLoginBody)
	}
	adminData := adminLoginBody["data"].(map[string]any)
	adminToken := adminData["access_token"].(string)

	// Fetch a valid Role for Siswa and Industri
	var siswaRoleIDStr, industriRoleIDStr, deptIDStr string
	_ = tc.db.Table("roles").Where("name = ?", "Siswa").Select("id::text").Scan(&siswaRoleIDStr).Error
	_ = tc.db.Table("roles").Where("name = ?", "Industri").Select("id::text").Scan(&industriRoleIDStr).Error
	siswaRoleID, _ := uuid.Parse(siswaRoleIDStr)
	industriRoleID, _ := uuid.Parse(industriRoleIDStr)

	// Fetch a Department (e.g. RPL)
	_ = tc.db.Table("departments").Where("code = ?", "RPL").Select("id::text").Scan(&deptIDStr).Error
	deptID, _ := uuid.Parse(deptIDStr)

	// Create a test Company
	testCompID := uuid.New()
	compSlug := fmt.Sprintf("pt-test-industri-%d", time.Now().UnixNano())
	_ = tc.db.Exec("INSERT INTO companies (id, name, slug, status, created_at, updated_at) VALUES (?, ?, ?, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
		testCompID, "PT Industri Mitra Test", compSlug).Error

	defer func() {
		_ = tc.db.Exec("DELETE FROM companies WHERE id = ?", testCompID).Error
	}()

	// 1. Admin creates an Akun Utama Industri (Industry Lead)
	leadEmail := fmt.Sprintf("lead-%d@mitra.com", time.Now().UnixNano())
	createLeadPayload := user.CreateUserRequest{
		Name:      "Lead PIC Industri",
		Email:     leadEmail,
		Password:  "IndustryPass123",
		RoleID:    industriRoleID,
		CompanyID: &testCompID,
		Status:    "active",
	}

	leadResp, leadBody, err := makeRequest(tc.app, "POST", "/api/v1/users", adminToken, createLeadPayload)
	if err != nil || leadResp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create industry lead user (status %d): %v", leadResp.StatusCode, leadBody)
	}

	leadData := leadBody["data"].(map[string]any)
	leadUserIDStr := leadData["id"].(string)
	leadUserID, _ := uuid.Parse(leadUserIDStr)

	defer func() {
		_ = tc.db.Exec("DELETE FROM users WHERE id = ?", leadUserID).Error
	}()

	// 2. Admin creates a Sub-Akun Siswa linked to this Company, Department, and Parent PIC
	studentEmail := fmt.Sprintf("siswa-%d@smkn6malang.sch.id", time.Now().UnixNano())
	createStudentPayload := user.CreateUserRequest{
		Name:         "Ahmad Siswa PKL",
		Email:        studentEmail,
		Password:     "SiswaPass123",
		RoleID:       siswaRoleID,
		DepartmentID: &deptID,
		CompanyID:    &testCompID,
		ParentID:     &leadUserID,
		Status:       "active",
	}

	studentResp, studentBody, err := makeRequest(tc.app, "POST", "/api/v1/users", adminToken, createStudentPayload)
	if err != nil || studentResp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create student user (status %d): %v", studentResp.StatusCode, studentBody)
	}

	studentData := studentBody["data"].(map[string]any)
	studentUserIDStr := studentData["id"].(string)
	studentUserID, _ := uuid.Parse(studentUserIDStr)

	defer func() {
		_ = tc.db.Exec("DELETE FROM users WHERE id = ?", studentUserID).Error
	}()

	// Verify Student Response has Department, Company, and ParentName
	if studentData["department_name"] == nil || studentData["company_name"] == nil {
		t.Errorf("Expected student to have department_name and company_name, got dept=%v, comp=%v",
			studentData["department_name"], studentData["company_name"])
	}
	if studentData["parent_name"] == nil {
		t.Errorf("Expected student to have parent_name set to lead, got nil")
	}

	// 3. Login as Industry Lead to verify scoping
	loginResp, loginBody, err := makeRequest(tc.app, "POST", "/api/v1/auth/login", "", map[string]any{
		"email":    leadEmail,
		"password": "IndustryPass123",
	})
	if err != nil || loginResp.StatusCode != http.StatusOK {
		t.Fatalf("Industry lead login failed: %v", err)
	}
	leadLoginData := loginBody["data"].(map[string]any)
	leadToken := leadLoginData["access_token"].(string)

	// 4. Industry Lead queries /api/v1/users
	// Should see their student and themselves, but NOT admin or other companies
	listResp, listBody, err := makeRequest(tc.app, "GET", "/api/v1/users", leadToken, nil)
	if err != nil || listResp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to list users as lead: %v", err)
	}

	items := listBody["data"].([]any)
	if len(items) != 2 {
		t.Errorf("Expected exactly 2 users in lead scope (lead + student), got %d", len(items))
	}
}
