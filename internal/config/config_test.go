package config_test

import (
	"strings"
	"testing"

	"pkl-golang-be/internal/config"
)

func TestLoadConfig_Development_UsesDefaults(t *testing.T) {
	// Lookup with no environment variables set
	mockLookup := func(key string) (string, bool) {
		return "", false
	}

	cfg, err := config.LoadConfigFromLookup(mockLookup)
	if err != nil {
		t.Fatalf("Expected LoadConfig to succeed in development with defaults, got error: %v", err)
	}

	if cfg.AppEnv != "development" {
		t.Errorf("Expected AppEnv 'development', got '%s'", cfg.AppEnv)
	}
	if cfg.IsProduction() {
		t.Errorf("Expected IsProduction() to be false")
	}
	if cfg.DBHost != "127.0.0.1" {
		t.Errorf("Expected default DBHost '127.0.0.1', got '%s'", cfg.DBHost)
	}
	if cfg.JWTSecret != config.DevDefaultJWTSecret {
		t.Errorf("Expected default JWTSecret, got '%s'", cfg.JWTSecret)
	}
}

func TestLoadConfig_Production_FailsOnMissingStrictKeys(t *testing.T) {
	// Mock production environment with missing strict variables
	envMap := map[string]string{
		"APP_ENV": "production",
		"PORT":    "8080",
	}

	mockLookup := func(key string) (string, bool) {
		val, ok := envMap[key]
		return val, ok
	}

	_, err := config.LoadConfigFromLookup(mockLookup)
	if err == nil {
		t.Fatalf("Expected error in production when strict variables are missing, got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "missing required environment variables") {
		t.Errorf("Expected error to mention missing required environment variables, got: %s", errMsg)
	}
	// Verify that critical missing keys are reported
	for _, expectedMissing := range []string{"DB_HOST", "DB_PASSWORD", "JWT_SECRET", "DEFAULT_ADMIN_PASSWORD"} {
		if !strings.Contains(errMsg, expectedMissing) {
			t.Errorf("Expected error to list missing key '%s', got: %s", expectedMissing, errMsg)
		}
	}
}

func TestLoadConfig_Production_FailsOnDefaultJWTSecret(t *testing.T) {
	// Mock production environment where JWT_SECRET is set to the dev placeholder
	envMap := map[string]string{
		"APP_ENV":                "production",
		"CORS_ALLOWED_ORIGINS":   "https://pkl.smkn6malang.sch.id",
		"DB_HOST":                "db.prod.internal",
		"DB_PORT":                "5432",
		"DB_USER":                "pkl_prod_user",
		"DB_PASSWORD":            "P@ssw0rdProdSecure!",
		"DB_NAME":                "pkl_production",
		"S3_ENDPOINT":            "https://s3.ap-southeast-1.amazonaws.com",
		"S3_BUCKET":              "pkl-production-bucket",
		"S3_ACCESS_KEY":          "PROD_ACCESS_KEY_XYZ",
		"S3_SECRET_KEY":          "PROD_SECRET_KEY_XYZ12345",
		"JWT_SECRET":             config.DevDefaultJWTSecret, // Insecure placeholder!
		"DEFAULT_ADMIN_NAME":     "Super Admin",
		"DEFAULT_ADMIN_EMAIL":    "admin@smkn6malang.sch.id",
		"DEFAULT_ADMIN_PASSWORD": "StrongAdminPass#2026",
	}

	mockLookup := func(key string) (string, bool) {
		val, ok := envMap[key]
		return val, ok
	}

	_, err := config.LoadConfigFromLookup(mockLookup)
	if err == nil {
		t.Fatalf("Expected error when JWT_SECRET uses development placeholder in production, got nil")
	}

	if !strings.Contains(err.Error(), "JWT_SECRET cannot use default development secret") {
		t.Errorf("Expected JWT placeholder error, got: %s", err.Error())
	}
}

func TestLoadConfig_Production_SucceedsWithAllFreshValues(t *testing.T) {
	envMap := map[string]string{
		"APP_ENV":                "production",
		"PORT":                   "4000",
		"CORS_ALLOWED_ORIGINS":   "https://pkl.smkn6malang.sch.id",
		"DB_HOST":                "db.prod.internal",
		"DB_PORT":                "5432",
		"DB_USER":                "pkl_prod_user",
		"DB_PASSWORD":            "P@ssw0rdProdSecure!",
		"DB_NAME":                "pkl_production",
		"S3_ENDPOINT":            "https://s3.ap-southeast-1.amazonaws.com",
		"S3_BUCKET":              "pkl-production-bucket",
		"S3_ACCESS_KEY":          "PROD_ACCESS_KEY_XYZ",
		"S3_SECRET_KEY":          "PROD_SECRET_KEY_XYZ12345",
		"JWT_SECRET":             "super_custom_production_secret_32_characters_long",
		"DEFAULT_ADMIN_NAME":     "Super Admin",
		"DEFAULT_ADMIN_EMAIL":    "admin@smkn6malang.sch.id",
		"DEFAULT_ADMIN_PASSWORD": "StrongAdminPass#2026",
	}

	mockLookup := func(key string) (string, bool) {
		val, ok := envMap[key]
		return val, ok
	}

	cfg, err := config.LoadConfigFromLookup(mockLookup)
	if err != nil {
		t.Fatalf("Expected LoadConfig to succeed with complete production env, got error: %v", err)
	}

	if !cfg.IsProduction() {
		t.Errorf("Expected IsProduction() to be true")
	}
	if cfg.Port != "4000" {
		t.Errorf("Expected Port '4000', got '%s'", cfg.Port)
	}
	if cfg.DBHost != "db.prod.internal" {
		t.Errorf("Expected DBHost 'db.prod.internal', got '%s'", cfg.DBHost)
	}
	if cfg.S3PublicEndpoint != cfg.S3Endpoint {
		t.Errorf("Expected S3PublicEndpoint to fallback to '%s', got '%s'", cfg.S3Endpoint, cfg.S3PublicEndpoint)
	}
}

func TestLoadConfig_Production_CustomS3PublicEndpoint(t *testing.T) {
	envMap := map[string]string{
		"APP_ENV":                "production",
		"PORT":                   "4000",
		"CORS_ALLOWED_ORIGINS":   "https://pkl.smkn6malang.sch.id",
		"DB_HOST":                "db.prod.internal",
		"DB_PORT":                "5432",
		"DB_USER":                "pkl_prod_user",
		"DB_PASSWORD":            "P@ssw0rdProdSecure!",
		"DB_NAME":                "pkl_production",
		"S3_ENDPOINT":            "http://pkl-storage:5000/s3",
		"S3_PUBLIC_ENDPOINT":     "https://pkl.smkn6malang.sch.id/s3",
		"S3_BUCKET":              "pkl-production-bucket",
		"S3_ACCESS_KEY":          "PROD_ACCESS_KEY_XYZ",
		"S3_SECRET_KEY":          "PROD_SECRET_KEY_XYZ12345",
		"JWT_SECRET":             "super_custom_production_secret_32_characters_long",
		"DEFAULT_ADMIN_NAME":     "Super Admin",
		"DEFAULT_ADMIN_EMAIL":    "admin@smkn6malang.sch.id",
		"DEFAULT_ADMIN_PASSWORD": "StrongAdminPass#2026",
	}

	mockLookup := func(key string) (string, bool) {
		val, ok := envMap[key]
		return val, ok
	}

	cfg, err := config.LoadConfigFromLookup(mockLookup)
	if err != nil {
		t.Fatalf("Expected LoadConfig to succeed, got error: %v", err)
	}

	if cfg.S3PublicEndpoint != "https://pkl.smkn6malang.sch.id/s3" {
		t.Errorf("Expected S3PublicEndpoint 'https://pkl.smkn6malang.sch.id/s3', got '%s'", cfg.S3PublicEndpoint)
	}
}
