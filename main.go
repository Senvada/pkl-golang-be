package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/infra/bootstrap"
	"pkl-golang-be/internal/infra/database"
	"pkl-golang-be/internal/infra/migrate"
	"pkl-golang-be/internal/infra/seeder"
	"pkl-golang-be/internal/infra/server"
	"pkl-golang-be/internal/infra/storage"
	"pkl-golang-be/internal/slices/auth"
	"pkl-golang-be/internal/slices/user"

	"gorm.io/gorm"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	command := "serve"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	switch command {
	case "serve":
		runServer(cfg)

	case "bootstrap":
		runBootstrap(cfg)

	case "seed":
		runSeed(cfg, os.Args[2:])

	case "migrate":
		runMigrate(cfg, os.Args[2:])

	case "test-db":
		runTestDB(cfg)

	case "test-s3":
		runTestS3(cfg)

	case "help", "--help", "-h":
		printHelp()

	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printHelp()
		os.Exit(1)
	}
}

func runServer(cfg *config.Config) {
	var db *gorm.DB
	var s3Client *storage.S3Client

	if cfg.AutoBootstrap {
		bootResult, err := bootstrap.Run(context.Background(), cfg)
		if err != nil {
			slog.Error("Auto-bootstrap failed", "error", err)
			os.Exit(1)
		}
		db = bootResult.DB
		s3Client = bootResult.S3Client
	} else {
		// Initialize Database
		var err error
		db, err = database.Connect(cfg)
		if err != nil {
			slog.Error("Failed to connect to database", "error", err)
			os.Exit(1)
		}

		// Auto-seed Default Admin (if not present, respecting soft-deleted accounts)
		if err := seeder.EnsureDefaultAdmin(db, cfg); err != nil {
			slog.Error("Failed to ensure default admin user", "error", err)
			os.Exit(1)
		}

		// Initialize Storage Client (S3)
		s3Client, err = storage.NewS3Client(cfg)
		if err != nil {
			slog.Warn("S3 storage client failed to initialize", "error", err)
		}
	}
	_ = s3Client // will be passed to slices later

	// Initialize and Run Server
	srv := server.New(cfg)

	// API v1 Router
	apiV1 := srv.App.Group("/api/v1")

	// Slices: Auth
	authService := auth.NewService(db, cfg)
	authHandler := auth.NewHandler(authService, cfg)
	auth.RegisterRoutes(apiV1, authHandler, cfg)

	// Slices: User
	userService := user.NewService(db, cfg, s3Client)
	userHandler := user.NewHandler(userService, cfg)
	user.RegisterRoutes(apiV1, userHandler, cfg)

	// API compatibility router for clients connecting via /api
	apiLegacy := srv.App.Group("/api")
	auth.RegisterRoutes(apiLegacy, authHandler, cfg)
	user.RegisterRoutes(apiLegacy, userHandler, cfg)

	if err := srv.Run(); err != nil {
		slog.Error("Server terminated unexpectedly", "error", err)
		os.Exit(1)
	}
}


func runMigrate(cfg *config.Config, args []string) {
	if len(args) == 0 {
		printMigrateHelp()
		os.Exit(1)
	}

	action := args[0]
	switch action {
	case "up":
		if err := migrate.Up(cfg); err != nil {
			slog.Error("Migration up failed", "error", err)
			os.Exit(1)
		}

	case "down":
		if len(args) > 1 {
			version, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				slog.Error("Invalid version argument", "input", args[1])
				os.Exit(1)
			}
			if err := migrate.DownToVersion(cfg, uint(version)); err != nil {
				slog.Error("Migration down to version failed", "error", err)
				os.Exit(1)
			}
		} else {
			if err := migrate.Down(cfg); err != nil {
				slog.Error("Migration down failed", "error", err)
				os.Exit(1)
			}
		}

	case "fresh":
		if err := migrate.Fresh(cfg); err != nil {
			slog.Error("Migration fresh failed", "error", err)
			os.Exit(1)
		}

	case "drop":
		if err := migrate.Drop(cfg); err != nil {
			slog.Error("Migration drop failed", "error", err)
			os.Exit(1)
		}

	case "version":
		ver, dirty, err := migrate.Version(cfg)
		if err != nil {
			slog.Error("Failed to get migration version", "error", err)
			os.Exit(1)
		}
		fmt.Printf("Current migration version: %d (dirty: %t)\n", ver, dirty)

	case "create":
		if len(args) < 2 {
			fmt.Println("Usage: go run main.go migrate create <migration_name>")
			os.Exit(1)
		}
		if err := migrate.Create(args[1]); err != nil {
			slog.Error("Failed to create migration", "error", err)
			os.Exit(1)
		}

	default:
		fmt.Printf("Unknown migration action: %s\n\n", action)
		printMigrateHelp()
		os.Exit(1)
	}
}

func runTestDB(cfg *config.Config) {
	slog.Info("Testing database connection...")
	db, err := database.Connect(cfg)
	if err != nil {
		slog.Error("❌ Database connection failed", "error", err)
		os.Exit(1)
	}
	slog.Info("✅ Database connection successful!")
	_ = db
}

func runTestS3(cfg *config.Config) {
	slog.Info("Testing S3 storage client...")
	s3Client, err := storage.NewS3Client(cfg)
	if err != nil {
		slog.Error("❌ S3 client initialization failed", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s3Client.EnsureBucket(ctx); err != nil {
		slog.Error("❌ S3 EnsureBucket failed", "error", err)
		os.Exit(1)
	}
	slog.Info("✅ S3 EnsureBucket verified successfully!")

	testKey := fmt.Sprintf("test/probe_%d.txt", time.Now().Unix())
	url, err := s3Client.PresignPutObject(ctx, testKey, "text/plain", 10, 5*time.Minute)
	if err != nil {
		slog.Error("❌ S3 PresignPutObject failed", "error", err)
		os.Exit(1)
	}

	slog.Info("✅ S3 Storage test successful!",
		"bucket", s3Client.Bucket(),
		"sample_presigned_url", url,
		"public_url_preview", s3Client.GetPublicURL(testKey),
	)
}

func runSeed(cfg *config.Config, args []string) {
	db, err := database.Connect(cfg)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	opts := seeder.Options{}
	for _, arg := range args {
		switch arg {
		case "--restore":
			opts.RestoreSoftDeleted = true
		case "--force-password", "--force":
			opts.ForceUpdatePassword = true
		}
	}

	slog.Info("Running database seeder...")
	if err := seeder.SeedDefaultAdmin(db, cfg, opts); err != nil {
		slog.Error("Default admin seeder failed", "error", err)
		os.Exit(1)
	}
	slog.Info("Database seeder finished successfully")
}

func runBootstrap(cfg *config.Config) {
	slog.Info("Running standalone environment bootstrap...")
	if _, err := bootstrap.Run(context.Background(), cfg); err != nil {
		slog.Error("Bootstrap execution failed", "error", err)
		os.Exit(1)
	}
	slog.Info("Bootstrap finished successfully!")
}

func printHelp() {
	fmt.Println(`PKL Backend API v2 (Vertical Slice Architecture)

Usage:
  go run main.go [command]

Available Commands:
  serve                 Start the Fiber HTTP API server (default, auto-bootstraps environment)
  bootstrap             Run full environment bootstrap (probe DB, ensure DB, migrate, seed admin, check storage)
  seed [flags]          Seed default admin user (--restore to restore soft-deleted, --force to update password)
  migrate [action]      Run database migrations (up, down, fresh, drop, version, create)
  test-db               Test connection to PostgreSQL database
  test-s3               Test S3 storage connection & generate sample presigned URL
  help                  Show this help message`)
}

func printMigrateHelp() {
	fmt.Println(`Migration commands:
  go run main.go migrate up                Apply all pending migrations
  go run main.go migrate down              Rollback 1 migration step
  go run main.go migrate down <version>    Rollback to specific version
  go run main.go migrate fresh             Drop all tables and re-apply all migrations
  go run main.go migrate drop              Drop all tables
  go run main.go migrate version           Show current migration version
  go run main.go migrate create <name>     Generate new migration .up.sql and .down.sql files`)
}
