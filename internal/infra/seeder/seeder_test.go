package seeder_test

import (
	"os"
	"testing"

	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/infra/database"
	"pkl-golang-be/internal/infra/seeder"
	"pkl-golang-be/internal/slices/auth"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestSeedDefaultAdmin_LifecycleAndSoftDelete(t *testing.T) {
	_ = godotenv.Load("../../../.env")
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	testEmail := "test-admin-softdelete@smkn6malang.sch.id"
	testPassword := "TestPass@123"

	// Cleanup any preexisting test user
	db.Unscoped().Where("email = ?", testEmail).Delete(&auth.User{})
	t.Cleanup(func() {
		db.Unscoped().Where("email = ?", testEmail).Delete(&auth.User{})
	})

	testCfg := *cfg
	testCfg.DefaultAdminEmail = testEmail
	testCfg.DefaultAdminPassword = testPassword
	testCfg.DefaultAdminName = "Test Seeder Admin"

	// 1. First run: should seed user
	err = seeder.EnsureDefaultAdmin(db, &testCfg)
	if err != nil {
		t.Fatalf("EnsureDefaultAdmin failed on initial seed: %v", err)
	}

	var seeded auth.User
	if err := db.Where("email = ?", testEmail).First(&seeded).Error; err != nil {
		t.Fatalf("Expected seeded user to exist, got error: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(seeded.Password), []byte(testPassword)); err != nil {
		t.Errorf("Password hash does not match test password: %v", err)
	}

	// 2. Second run: idempotent when user is active
	err = seeder.EnsureDefaultAdmin(db, &testCfg)
	if err != nil {
		t.Fatalf("EnsureDefaultAdmin failed on second run: %v", err)
	}

	var count int64
	db.Unscoped().Model(&auth.User{}).Where("email = ?", testEmail).Count(&count)
	if count != 1 {
		t.Fatalf("Expected exactly 1 user, found %d", count)
	}

	// 3. Soft-delete the user
	if err := db.Delete(&seeded).Error; err != nil {
		t.Fatalf("Failed to soft-delete test user: %v", err)
	}

	// Verify it's soft-deleted
	var activeCheck auth.User
	if err := db.Where("email = ?", testEmail).First(&activeCheck).Error; err != gorm.ErrRecordNotFound {
		t.Fatalf("Expected soft-deleted user to NOT be found in normal query, got err: %v", err)
	}

	// 4. Third run: EnsureDefaultAdmin MUST NOT re-seed the soft-deleted user
	err = seeder.EnsureDefaultAdmin(db, &testCfg)
	if err != nil {
		t.Fatalf("EnsureDefaultAdmin failed when checking soft-deleted user: %v", err)
	}

	// Verify still exactly 1 record in database and it remains soft-deleted
	db.Unscoped().Model(&auth.User{}).Where("email = ?", testEmail).Count(&count)
	if count != 1 {
		t.Fatalf("Expected count to remain 1 (no duplicate insertion on soft-deleted account), got %d", count)
	}

	var softDeletedUser auth.User
	if err := db.Unscoped().Where("email = ?", testEmail).First(&softDeletedUser).Error; err != nil {
		t.Fatalf("Expected soft-deleted user to exist in unscoped query: %v", err)
	}
	if !softDeletedUser.DeletedAt.Valid {
		t.Fatalf("Expected user to remain soft-deleted (deleted_at IS NOT NULL)")
	}

	// 5. Test CLI restore flag (--restore)
	err = seeder.SeedDefaultAdmin(db, &testCfg, seeder.Options{RestoreSoftDeleted: true})
	if err != nil {
		t.Fatalf("SeedDefaultAdmin with RestoreSoftDeleted failed: %v", err)
	}

	var restoredUser auth.User
	if err := db.Where("email = ?", testEmail).First(&restoredUser).Error; err != nil {
		t.Fatalf("Expected restored user to be found in normal query, got err: %v", err)
	}
	if restoredUser.DeletedAt.Valid {
		t.Fatalf("Expected restored user to have null deleted_at")
	}
	if restoredUser.Status != "active" {
		t.Fatalf("Expected restored user to have active status, got %s", restoredUser.Status)
	}
}

func TestSeederSQL_Execution(t *testing.T) {
	_ = godotenv.Load("../../../.env")
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	db, err := database.Connect(cfg)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	sqlBytes, err := os.ReadFile("../../../database/seeder.sql")
	if err != nil {
		t.Fatalf("Failed to read seeder.sql: %v", err)
	}

	if err := db.Exec(string(sqlBytes)).Error; err != nil {
		t.Fatalf("Failed to execute seeder.sql: %v", err)
	}
	var count int64
	err = db.Table("assessment_templates").
		Joins("JOIN academic_periods ON academic_periods.id = assessment_templates.period_id").
		Where("academic_periods.name = ? AND assessment_templates.deleted_at IS NULL", "2025/2026").
		Count(&count).Error
	if err != nil {
		t.Fatalf("Failed to count assessment templates: %v", err)
	}
	if count != 8 {
		t.Fatalf("Expected exactly 8 seeded templates (without SIJA & KJIJ), got %d", count)
	}
}
