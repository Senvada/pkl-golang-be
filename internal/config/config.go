package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const DevDefaultJWTSecret = "super_secret_jwt_key_pkl_development_minimum_32_chars"

type Config struct {
	AppEnv             string
	Port               string
	CORSAllowedOrigins string

	// Database
	DBHost            string
	DBPort            string
	DBUser            string
	DBPassword        string
	DBName            string
	DBSSLMode         string
	DBTimeZone        string
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime int

	// Storage (S3 / Supabase Storage)
	S3Endpoint        string
	S3Region          string
	S3Bucket          string
	S3AccessKey       string
	S3SecretKey       string
	S3PublicURLPrefix string
	S3UsePathStyle    bool

	// Auth
	JWTSecret      string
	JWTExpiryHours int
	GoogleClientID string

	// Default Admin Seeder
	DefaultAdminName     string
	DefaultAdminEmail    string
	DefaultAdminPassword string
}

func (c *Config) IsProduction() bool {
	return strings.EqualFold(c.AppEnv, "production")
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		slog.Debug(".env file not found, using system environment variables")
	}

	return LoadConfigFromLookup(os.LookupEnv)
}

func LoadConfigFromLookup(lookup func(string) (string, bool)) (*Config, error) {
	appEnvVal, _ := lookup("APP_ENV")
	appEnv := strings.ToLower(strings.TrimSpace(appEnvVal))
	if appEnv == "" {
		appEnv = "development"
	}
	isProd := appEnv == "production"

	loader := &envLoader{
		isProd: isProd,
		lookup: lookup,
	}

	cfg := &Config{
		AppEnv:             appEnv,
		Port:               loader.getSafe("PORT", "3000"),
		CORSAllowedOrigins: loader.getStrict("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:3000"),

		// Database (Strict in production: host, port, credentials, and dbname must be explicitly provided)
		DBHost:            loader.getStrict("DB_HOST", "127.0.0.1"),
		DBPort:            loader.getStrict("DB_PORT", "54322"),
		DBUser:            loader.getStrict("DB_USER", "postgres"),
		DBPassword:        loader.getStrict("DB_PASSWORD", "postgres"),
		DBName:            loader.getStrict("DB_NAME", "pkl_development"),
		DBSSLMode:         loader.getSafe("DB_SSLMODE", "disable"),
		DBTimeZone:        loader.getSafe("DB_TIMEZONE", "Asia/Jakarta"),
		DBMaxOpenConns:    loader.getSafeInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:    loader.getSafeInt("DB_MAX_IDLE_CONNS", 10),
		DBConnMaxLifetime: loader.getSafeInt("DB_CONN_MAX_LIFETIME", 300),

		// Storage (Strict in production: endpoint, credentials, and bucket must be explicitly provided)
		S3Endpoint:        loader.getStrict("S3_ENDPOINT", "http://127.0.0.1:54321/storage/v1/s3"),
		S3Region:          loader.getSafe("S3_REGION", "local"),
		S3Bucket:          loader.getStrict("S3_BUCKET", "pkl-assets"),
		S3AccessKey:       loader.getStrict("S3_ACCESS_KEY", "625729a08b95bf1b7ff351a663f3a23c"),
		S3SecretKey:       loader.getStrict("S3_SECRET_KEY", "850181e4652dd023b7a98c58ae0d2d34bd487ee0cc3254aed6eda37307425907"),
		S3PublicURLPrefix: loader.getSafe("S3_PUBLIC_URL_PREFIX", "http://127.0.0.1:54321/storage/v1/object/public/pkl-assets"),
		S3UsePathStyle:    loader.getSafeBool("S3_USE_PATH_STYLE", true),

		// Auth
		JWTSecret:      loader.getStrict("JWT_SECRET", DevDefaultJWTSecret),
		JWTExpiryHours: loader.getSafeInt("JWT_EXPIRY_HOURS", 72),
		GoogleClientID: loader.getSafe("GOOGLE_CLIENT_ID", ""),

		// Default Admin Seeder (Strict in production: admin credentials must be explicitly configured)
		DefaultAdminName:     loader.getStrict("DEFAULT_ADMIN_NAME", "Administrator PKL"),
		DefaultAdminEmail:    loader.getStrict("DEFAULT_ADMIN_EMAIL", "admin@smkn6malang.sch.id"),
		DefaultAdminPassword: loader.getStrict("DEFAULT_ADMIN_PASSWORD", "PKL@2026"),
	}

	if isProd {
		if len(loader.missingKeys) > 0 {
			return nil, fmt.Errorf("production configuration error: missing required environment variables: [%s]", strings.Join(loader.missingKeys, ", "))
		}
		if cfg.JWTSecret == DevDefaultJWTSecret {
			return nil, fmt.Errorf("production configuration error: JWT_SECRET cannot use default development secret")
		}
	}

	return cfg, nil
}

func (c *Config) PostgresDSN() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		c.DBHost, c.DBUser, c.DBPassword, c.DBName, c.DBPort, c.DBSSLMode, c.DBTimeZone,
	)
}

func (c *Config) MigrateDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode,
	)
}

type envLoader struct {
	isProd      bool
	lookup      func(string) (string, bool)
	missingKeys []string
}

// getStrict reads an environment variable.
// In development, it allows fallback to devDefault if the variable is not set.
// In production, the variable MUST be explicitly set and non-empty;
// otherwise, it records the missing key and returns an empty string.
func (l *envLoader) getStrict(key, devDefault string) string {
	val, ok := l.lookup(key)
	val = strings.TrimSpace(val)
	if ok && val != "" {
		return val
	}

	if l.isProd {
		l.missingKeys = append(l.missingKeys, key)
		return ""
	}
	return devDefault
}

// getSafe reads an environment variable with a safe default permitted across all environments.
func (l *envLoader) getSafe(key, defaultVal string) string {
	val, ok := l.lookup(key)
	val = strings.TrimSpace(val)
	if ok && val != "" {
		return val
	}
	return defaultVal
}

func (l *envLoader) getSafeInt(key string, defaultVal int) int {
	valStr := l.getSafe(key, "")
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return defaultVal
}

func (l *envLoader) getSafeBool(key string, defaultVal bool) bool {
	valStr := l.getSafe(key, "")
	if val, err := strconv.ParseBool(valStr); err == nil {
		return val
	}
	return defaultVal
}
