package migrate

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"pkl-golang-be/internal/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const migrationsDir = "database/migrations"

func newMigrator(cfg *config.Config) (*migrate.Migrate, error) {
	dsn := cfg.MigrateDSN()
	sourceURL := fmt.Sprintf("file://%s", migrationsDir)

	m, err := migrate.New(sourceURL, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return m, nil
}

func Up(cfg *config.Config) error {
	m, err := newMigrator(cfg)
	if err != nil {
		return err
	}
	defer m.Close()

	slog.Info("Running migrations Up...")
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("No new migrations to apply.")
			return nil
		}
		return fmt.Errorf("migration Up failed: %w", err)
	}

	slog.Info("Migration Up completed successfully.")
	return nil
}

func Down(cfg *config.Config) error {
	m, err := newMigrator(cfg)
	if err != nil {
		return err
	}
	defer m.Close()

	slog.Info("Rolling back 1 migration step...")
	if err := m.Steps(-1); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("No migrations to rollback.")
			return nil
		}
		return fmt.Errorf("migration Down failed: %w", err)
	}

	slog.Info("Migration step rolled back successfully.")
	return nil
}

func DownToVersion(cfg *config.Config, version uint) error {
	m, err := newMigrator(cfg)
	if err != nil {
		return err
	}
	defer m.Close()

	slog.Info("Migrating to specific version...", "version", version)
	if err := m.Migrate(version); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("Database is already at target version.")
			return nil
		}
		return fmt.Errorf("migration to version %d failed: %w", version, err)
	}

	slog.Info("Migrated to version successfully.", "version", version)
	return nil
}

func Drop(cfg *config.Config) error {
	m, err := newMigrator(cfg)
	if err != nil {
		return err
	}
	defer m.Close()

	slog.Warn("Dropping all database tables and types...")
	if err := m.Drop(); err != nil {
		return fmt.Errorf("migration Drop failed: %w", err)
	}

	slog.Info("All database tables dropped.")
	return nil
}

func Fresh(cfg *config.Config) error {
	slog.Warn("Refreshing database (Drop + Up)...")
	if err := Drop(cfg); err != nil {
		return err
	}
	return Up(cfg)
}

func Version(cfg *config.Config) (uint, bool, error) {
	m, err := newMigrator(cfg)
	if err != nil {
		return 0, false, err
	}
	defer m.Close()

	version, dirty, err := m.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			return 0, false, nil
		}
		return 0, false, err
	}

	return version, dirty, nil
}

func Create(name string) error {
	if name == "" {
		return errors.New("migration name cannot be empty")
	}

	timestamp := time.Now().Format("20060102150405")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	upFile := filepath.Join(migrationsDir, fmt.Sprintf("%s_%s.up.sql", timestamp, name))
	downFile := filepath.Join(migrationsDir, fmt.Sprintf("%s_%s.down.sql", timestamp, name))

	if err := os.WriteFile(upFile, []byte(fmt.Sprintf("-- Migration: %s (up)\n", name)), 0644); err != nil {
		return fmt.Errorf("failed to create up migration file: %w", err)
	}

	if err := os.WriteFile(downFile, []byte(fmt.Sprintf("-- Migration: %s (down)\n", name)), 0644); err != nil {
		return fmt.Errorf("failed to create down migration file: %w", err)
	}

	slog.Info("Created new migration files:",
		"up", upFile,
		"down", downFile,
	)
	return nil
}
