package seeder

import (
	"errors"
	"fmt"
	"log/slog"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/slices/auth"
)

type Options struct {
	RestoreSoftDeleted  bool
	ForceUpdatePassword bool
}

// EnsureDefaultAdmin checks if the default admin user exists in the database.
// It uses an Unscoped query to check both active and soft-deleted records.
// If the record exists (active or deleted), it will NOT re-seed to prevent repeated seeding of deleted accounts.
func EnsureDefaultAdmin(db *gorm.DB, cfg *config.Config) error {
	if err := EnsureDefaultRolesAndDepartments(db); err != nil {
		slog.Warn("Failed to ensure default roles and departments", "error", err)
	}
	return SeedDefaultAdmin(db, cfg, Options{})
}

// SeedRoles seeds initial roles if not yet created.
func SeedRoles(db *gorm.DB) error {
	type defaultRole struct {
		Name  string
		Color string
		Perm  string
	}

	roles := []defaultRole{
		{Name: "Admin", Color: "#EF4444", Perm: `{"admin": true}`},
		{Name: "Industri", Color: "#8B5CF6", Perm: `{"company": {"read": true, "update": true}, "users": {"read": true, "create": true, "update": true}, "article": {"read": true, "create": true, "update": true}}`},
		{Name: "Guru", Color: "#10B981", Perm: `{"users": {"read": true}, "company": {"read": true}, "article": {"read": true}}`},
		{Name: "Pembimbing", Color: "#F59E0B", Perm: `{"users": {"read": true}, "company": {"read": true}, "article": {"read": true}}`},
		{Name: "Siswa", Color: "#06B6D4", Perm: `{"article": {"read": true, "create": true, "update": true}, "company": {"read": true}}`},
	}

	for _, r := range roles {
		var count int64
		if err := db.Table("roles").Where("name = ? AND deleted_at IS NULL", r.Name).Count(&count).Error; err == nil && count == 0 {
			if err := db.Exec("INSERT INTO roles (id, name, color, permissions, created_at, updated_at) VALUES (gen_random_uuid(), ?, ?, ?::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
				r.Name, r.Color, r.Perm).Error; err != nil {
				return fmt.Errorf("failed to insert default role %s: %w", r.Name, err)
			}
			slog.Info("✅ Seeded default role", "name", r.Name)
		}
	}

	return nil
}

// SeedDepartments seeds the 10 default vocational departments (jurusan).
func SeedDepartments(db *gorm.DB) error {
	type defaultDept struct {
		Name  string
		Code  string
		Color string
	}

	depts := []defaultDept{
		{Name: "Rekayasa Perangkat Lunak", Code: "RPL", Color: "#3B82F6"},
		{Name: "Teknik Komputer dan Jaringan", Code: "TKJ", Color: "#10B981"},
		{Name: "Sistem Informasi, Jaringan, dan Aplikasi", Code: "SIJA", Color: "#6366F1"},
		{Name: "Desain Pemodelan dan Informasi Bangunan", Code: "DPIB", Color: "#EC4899"},
		{Name: "Teknik Pemesinan", Code: "TPM", Color: "#F97316"},
		{Name: "Teknik Kendaraan Ringan", Code: "TKR", Color: "#EF4444"},
		{Name: "Teknik Instalasi Tenaga Listrik", Code: "TITL", Color: "#F59E0B"},
		{Name: "Teknik Alat Berat", Code: "TAB", Color: "#84CC16"},
		{Name: "Teknik Otomotif", Code: "OTO", Color: "#14B8A6"},
		{Name: "Konstruksi Jalan, Irigasi, dan Jembatan", Code: "KJIJ", Color: "#8B5CF6"},
	}

	for _, d := range depts {
		var count int64
		if err := db.Table("departments").Where("code = ? AND deleted_at IS NULL", d.Code).Count(&count).Error; err == nil && count == 0 {
			if err := db.Exec("INSERT INTO departments (id, name, code, color, created_at, updated_at) VALUES (gen_random_uuid(), ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
				d.Name, d.Code, d.Color).Error; err != nil {
				return fmt.Errorf("failed to insert default department %s: %w", d.Code, err)
			}
			slog.Info("✅ Seeded default department", "code", d.Code, "name", d.Name)
		}
	}

	return nil
}

// EnsureDefaultRolesAndDepartments seeds initial roles and departments if not yet created.
func EnsureDefaultRolesAndDepartments(db *gorm.DB) error {
	if err := SeedRoles(db); err != nil {
		return err
	}
	return SeedDepartments(db)
}

// SeedDefaultAdmin seeds the default admin user with configurable options.
func SeedDefaultAdmin(db *gorm.DB, cfg *config.Config, opts Options) error {
	if cfg.DefaultAdminEmail == "" || cfg.DefaultAdminPassword == "" {
		slog.Warn("Default admin email or password is empty; skipping default admin seeder")
		return nil
	}

	// 1. Ensure Admin Role exists
	var adminRole auth.Role
	err := db.Unscoped().Where("name = ?", "Admin").First(&adminRole).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		adminRole = auth.Role{
			Name:        "Admin",
			Color:       "#EF4444",
			Permissions: datatypes.JSON([]byte(`{"admin": true}`)),
		}
		if err := db.Create(&adminRole).Error; err != nil {
			return fmt.Errorf("failed to create default Admin role: %w", err)
		}
		slog.Info("Created default Admin role", "role_id", adminRole.ID)
	} else if err != nil {
		return fmt.Errorf("failed to query Admin role: %w", err)
	}

	// 2. Check if default admin user already exists (active or soft-deleted)
	var existing auth.User
	err = db.Unscoped().Where("email = ?", cfg.DefaultAdminEmail).First(&existing).Error
	if err == nil {
		// User already exists in database
		if existing.DeletedAt.Valid {
			if opts.RestoreSoftDeleted {
				slog.Info("Restoring soft-deleted default admin user as requested", "email", cfg.DefaultAdminEmail)
				updates := map[string]interface{}{
					"deleted_at": nil,
					"status":     "active",
				}
				if opts.ForceUpdatePassword {
					hashedPassword, hashErr := bcrypt.GenerateFromPassword([]byte(cfg.DefaultAdminPassword), bcrypt.DefaultCost)
					if hashErr != nil {
						return fmt.Errorf("failed to hash default admin password: %w", hashErr)
					}
					updates["password"] = string(hashedPassword)
				}
				if err := db.Unscoped().Model(&existing).Updates(updates).Error; err != nil {
					return fmt.Errorf("failed to restore soft-deleted admin user: %w", err)
				}
				slog.Info("✅ Default admin user successfully restored", "email", cfg.DefaultAdminEmail)
				return nil
			}

			slog.Info("Default admin user already exists in database as soft-deleted record. Skipping auto-seed to respect deletion.",
				"email", cfg.DefaultAdminEmail,
				"deleted_at", existing.DeletedAt.Time,
			)
			return nil
		}

		// User is currently active
		if opts.ForceUpdatePassword {
			slog.Info("Force updating default admin password as requested", "email", cfg.DefaultAdminEmail)
			hashedPassword, hashErr := bcrypt.GenerateFromPassword([]byte(cfg.DefaultAdminPassword), bcrypt.DefaultCost)
			if hashErr != nil {
				return fmt.Errorf("failed to hash default admin password: %w", hashErr)
			}
			if err := db.Model(&existing).Update("password", string(hashedPassword)).Error; err != nil {
				return fmt.Errorf("failed to update default admin password: %w", err)
			}
			slog.Info("✅ Default admin password successfully updated", "email", cfg.DefaultAdminEmail)
			return nil
		}

		slog.Debug("Default admin user already exists. Auto-seed skipped.",
			"email", cfg.DefaultAdminEmail,
			"id", existing.ID,
		)
		return nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to query existing default admin user: %w", err)
	}

	// 3. User does not exist at all -> Hash password and insert
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(cfg.DefaultAdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash default admin password: %w", err)
	}

	adminUser := auth.User{
		RoleID:   adminRole.ID,
		Name:     cfg.DefaultAdminName,
		Email:    cfg.DefaultAdminEmail,
		Password: string(hashedPassword),
		Status:   "active",
	}

	if err := db.Create(&adminUser).Error; err != nil {
		return fmt.Errorf("failed to insert default admin user: %w", err)
	}

	slog.Info("✅ Default admin user successfully seeded",
		"email", adminUser.Email,
		"name", adminUser.Name,
		"role", "Admin",
	)
	return nil
}
