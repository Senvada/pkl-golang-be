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
	return SeedDefaultAdmin(db, cfg, Options{})
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
