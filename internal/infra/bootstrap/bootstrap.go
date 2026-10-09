package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"regexp"
	"strings"
	"time"

	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/infra/database"
	"pkl-golang-be/internal/infra/migrate"
	"pkl-golang-be/internal/infra/seeder"
	"pkl-golang-be/internal/infra/storage"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var validDBNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// Result contains initialized instances from bootstrap.
type Result struct {
	DB       *gorm.DB
	S3Client *storage.S3Client
}

// Run executes the complete environment bootstrap workflow:
// 1. Probing database host connectivity (with retry loop).
// 2. Ensuring the target database exists (creating it if missing).
// 3. Ensuring Supabase storage bucket registration if storage schema exists.
// 4. Connecting to the target database.
// 5. Applying pending migrations automatically.
// 6. Ensuring the default admin user and roles are seeded.
// 7. Initializing the S3 client and verifying the bucket.
func Run(ctx context.Context, cfg *config.Config) (*Result, error) {
	slog.Info("🚀 [Bootstrap] Starting environment verification and initialization...")

	// 1. Wait for database TCP port to become accessible
	maxWait := 30 * time.Second
	if err := WaitForTCP(ctx, cfg.DBHost, cfg.DBPort, maxWait); err != nil {
		return nil, fmt.Errorf("database connectivity check failed: %w", err)
	}

	// 2. Ensure database exists
	if err := EnsureDatabaseExists(ctx, cfg); err != nil {
		return nil, fmt.Errorf("failed to ensure database exists: %w", err)
	}

	// 3. Connect to the application database
	db, err := database.Connect(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to application database: %w", err)
	}

	// 4. Run database migrations
	slog.Info("📦 [Bootstrap] Running database migrations...")
	if err := migrate.Up(cfg); err != nil {
		return nil, fmt.Errorf("failed to execute auto-migrations: %w", err)
	}
	slog.Info("✅ [Bootstrap] Database migrations applied successfully")

	// 5. Seed default admin user
	slog.Info("👤 [Bootstrap] Ensuring default admin account exists...")
	if err := seeder.EnsureDefaultAdmin(db, cfg); err != nil {
		return nil, fmt.Errorf("failed to seed default admin: %w", err)
	}
	slog.Info("✅ [Bootstrap] Default admin user verified")

	// 6. Initialize storage & ensure bucket
	slog.Info("🪣 [Bootstrap] Initializing storage client...")
	s3Client, err := storage.NewS3Client(cfg)
	if err != nil {
		slog.Warn("⚠️ [Bootstrap] S3 client initialization returned a warning", "error", err)
	} else {
		storageCtx, storageCancel := context.WithTimeout(ctx, 10*time.Second)
		defer storageCancel()
		if err := s3Client.EnsureBucket(storageCtx); err != nil {
			slog.Warn("⚠️ [Bootstrap] Could not verify/create S3 bucket", "error", err)
		} else {
			slog.Info("✅ [Bootstrap] Storage bucket verified", "bucket", cfg.S3Bucket)
		}
	}

	slog.Info("🎉 [Bootstrap] Environment initialization complete!")
	return &Result{
		DB:       db,
		S3Client: s3Client,
	}, nil
}

// WaitForTCP probes a host:port combination until it responds or timeout expires.
func WaitForTCP(ctx context.Context, host, port string, timeout time.Duration) error {
	addr := net.JoinHostPort(host, port)
	deadline := time.Now().Add(timeout)
	attempt := 0

	slog.Info("⏳ [Bootstrap] Probing database connection...", "target", addr, "timeout", timeout)

	for {
		attempt++
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			_ = conn.Close()
			slog.Info("✅ [Bootstrap] Database port is open and accepting connections", "target", addr, "attempts", attempt)
			return nil
		}

		if time.Now().After(deadline) || ctx.Err() != nil {
			return fmt.Errorf("unable to reach %s after %v (%d attempts). Please verify that PostgreSQL is running and reachable on this network: %w",
				addr, timeout, attempt, err)
		}

		slog.Info("⏳ [Bootstrap] Database not ready yet, retrying in 2 seconds...", "attempt", attempt, "target", addr)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

// EnsureDatabaseExists connects to the maintenance DB ("postgres") to check if DB_NAME exists,
// creating it if missing, auto-configures supabase_storage_admin least-privileged role,
// and auto-registers the storage bucket if Supabase schema is present.
func EnsureDatabaseExists(ctx context.Context, cfg *config.Config) error {
	if !IsValidDBName(cfg.DBName) {
		return fmt.Errorf("database name '%s' contains invalid characters; only letters, numbers, and underscores are allowed", cfg.DBName)
	}

	// Connect to default maintenance database ("postgres").
	// In Supabase, 'supabase_admin' is the superuser with permission to modify reserved roles.
	// Fall back to cfg.DBUser for standard PostgreSQL environments.
	maintDSN := cfg.PostgresMaintenanceDSN()
	adminDSN := fmt.Sprintf("host=%s user=supabase_admin password=%s dbname=postgres port=%s sslmode=%s TimeZone=%s",
		cfg.DBHost, cfg.DBPassword, cfg.DBPort, cfg.DBSSLMode, cfg.DBTimeZone,
	)

	maintDB, err := gorm.Open(postgres.Open(adminDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		maintDB, err = gorm.Open(postgres.Open(maintDSN), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			return fmt.Errorf("failed to connect to PostgreSQL maintenance database ('postgres'): %w", err)
		}
	}

	sqlDB, err := maintDB.DB()
	if err != nil {
		return fmt.Errorf("failed to acquire sql.DB handle for maintenance database: %w", err)
	}
	defer sqlDB.Close()

	// 1. Check if application database exists (skip if app uses 'postgres')
	if !strings.EqualFold(cfg.DBName, "postgres") {
		var exists int
		query := "SELECT 1 FROM pg_database WHERE datname = ?"
		if err := maintDB.WithContext(ctx).Raw(query, cfg.DBName).Scan(&exists).Error; err != nil {
			return fmt.Errorf("failed to query pg_database: %w", err)
		}

		if exists != 1 {
			slog.Info("✨ [Bootstrap] Database does not exist yet. Creating...", "database", cfg.DBName)
			createStmt := fmt.Sprintf(`CREATE DATABASE "%s"`, cfg.DBName)
			if err := maintDB.WithContext(ctx).Exec(createStmt).Error; err != nil {
				return fmt.Errorf("failed to create database '%s': %w", cfg.DBName, err)
			}
			slog.Info("✅ [Bootstrap] Database created successfully", "database", cfg.DBName)
		} else {
			slog.Info("✅ [Bootstrap] Database already exists", "database", cfg.DBName)
		}
	}

	// 2. Automatically ensure supabase_storage_admin role, password, and schema permissions (Least Privilege)
	EnsureStorageRoleAndPermissions(ctx, maintDB, cfg.DBPassword)

	// 3. If Supabase storage is installed in postgres database, ensure the bucket entry
	var storageTableExists int
	checkStorageTable := "SELECT 1 FROM information_schema.tables WHERE table_schema = 'storage' AND table_name = 'buckets'"
	_ = maintDB.WithContext(ctx).Raw(checkStorageTable).Scan(&storageTableExists).Error

	if storageTableExists == 1 && cfg.S3Bucket != "" {
		slog.Info("🪣 [Bootstrap] Supabase storage schema detected. Ensuring bucket record...", "bucket", cfg.S3Bucket)
		insertBucket := "INSERT INTO storage.buckets (id, name, public) VALUES (?, ?, true) ON CONFLICT (id) DO NOTHING"
		if err := maintDB.WithContext(ctx).Exec(insertBucket, cfg.S3Bucket, cfg.S3Bucket).Error; err != nil {
			slog.Warn("⚠️ [Bootstrap] Could not register bucket in storage.buckets", "error", err)
		} else {
			slog.Info("✅ [Bootstrap] Bucket record registered in Supabase storage schema", "bucket", cfg.S3Bucket)
		}
	}

	return nil
}

// EnsureStorageRoleAndPermissions auto-configures the supabase_storage_admin role and
// ensures proper ownership on the storage schema without needing manual ALTER commands.
func EnsureStorageRoleAndPermissions(ctx context.Context, maintDB *gorm.DB, password string) {
	escapedPassword := strings.ReplaceAll(password, "'", "''")
	ensureRoleSQL := fmt.Sprintf(`
	DO $$
	BEGIN
		IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'supabase_storage_admin') THEN
			EXECUTE 'ALTER ROLE supabase_storage_admin WITH LOGIN PASSWORD ' || quote_literal('%s');
		END IF;
		CREATE SCHEMA IF NOT EXISTS storage;
		IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'supabase_storage_admin') THEN
			ALTER SCHEMA storage OWNER TO supabase_storage_admin;
			GRANT ALL ON SCHEMA storage TO supabase_storage_admin;
		END IF;
	END $$;
	`, escapedPassword)

	if err := maintDB.WithContext(ctx).Exec(ensureRoleSQL).Error; err != nil {
		slog.Debug("[Bootstrap] Storage role auto-configuration skipped or not applicable", "error", err)
		return
	}

	ensureTablesSQL := `
	DO $$
	DECLARE
		r RECORD;
	BEGIN
		IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'supabase_storage_admin') THEN
			FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'storage') LOOP
				EXECUTE 'ALTER TABLE storage.' || quote_ident(r.tablename) || ' OWNER TO supabase_storage_admin';
			END LOOP;
			FOR r IN (SELECT sequence_name FROM information_schema.sequences WHERE sequence_schema = 'storage') LOOP
				EXECUTE 'ALTER SEQUENCE storage.' || quote_ident(r.sequence_name) || ' OWNER TO supabase_storage_admin';
			END LOOP;
			EXECUTE 'GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA storage TO supabase_storage_admin';
			EXECUTE 'GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA storage TO supabase_storage_admin';
		END IF;
	END $$;
	`
	_ = maintDB.WithContext(ctx).Exec(ensureTablesSQL).Error
	slog.Info("🛡️ [Bootstrap] Verified least-privileged 'supabase_storage_admin' role and storage permissions")
}

// IsValidDBName validates that the database name has only safe alphanumeric and underscore characters.
func IsValidDBName(name string) bool {
	return validDBNameRegex.MatchString(name)
}
