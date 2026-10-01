package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

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
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		slog.Info(".env file not found, using system environment variables")
	}

	cfg := &Config{
		AppEnv:             getEnv("APP_ENV", "development"),
		Port:               getEnv("PORT", "3000"),
		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "*"),

		// Database defaults (Supabase Local Postgres)
		DBHost:            getEnv("DB_HOST", "127.0.0.1"),
		DBPort:            getEnv("DB_PORT", "54322"),
		DBUser:            getEnv("DB_USER", "postgres"),
		DBPassword:        getEnv("DB_PASSWORD", "postgres"),
		DBName:            getEnv("DB_NAME", "pkl_development"),
		DBSSLMode:         getEnv("DB_SSLMODE", "disable"),
		DBTimeZone:        getEnv("DB_TIMEZONE", "Asia/Jakarta"),
		DBMaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
		DBConnMaxLifetime: getEnvAsInt("DB_CONN_MAX_LIFETIME", 300),

		// Storage defaults (Supabase Local Storage S3)
		S3Endpoint:        getEnv("S3_ENDPOINT", "http://127.0.0.1:54321/storage/v1/s3"),
		S3Region:          getEnv("S3_REGION", "local"),
		S3Bucket:          getEnv("S3_BUCKET", "pkl-assets"),
		S3AccessKey:       getEnv("S3_ACCESS_KEY", "625729a08b95bf1b7ff351a663f3a23c"),
		S3SecretKey:       getEnv("S3_SECRET_KEY", "850181e4652dd023b7a98c58ae0d2d34bd487ee0cc3254aed6eda37307425907"),
		S3PublicURLPrefix: getEnv("S3_PUBLIC_URL_PREFIX", "http://127.0.0.1:54321/storage/v1/object/public/pkl-assets"),
		S3UsePathStyle:    getEnvAsBool("S3_USE_PATH_STYLE", true),

		// Auth
		JWTSecret:      getEnv("JWT_SECRET", "super_secret_jwt_key_pkl_development_minimum_32_chars"),
		JWTExpiryHours: getEnvAsInt("JWT_EXPIRY_HOURS", 72),
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

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	valStr := getEnv(key, "")
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return defaultVal
}

func getEnvAsBool(key string, defaultVal bool) bool {
	valStr := getEnv(key, "")
	if val, err := strconv.ParseBool(valStr); err == nil {
		return val
	}
	return defaultVal
}
