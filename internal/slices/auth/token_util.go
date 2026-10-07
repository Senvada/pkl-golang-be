package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"pkl-golang-be/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	RefreshTokenCookieName = "pkl-refresh-token"
	AccessTokenDuration    = 15 * time.Minute
	RefreshTokenDuration   = 24 * time.Hour
	RememberMeDuration     = 30 * 24 * time.Hour
)

func GenerateAccessToken(user *User, roleName string, secret string) (string, int64, error) {
	expiresAt := time.Now().Add(AccessTokenDuration)

	claims := middleware.AuthClaims{
		UserID:    user.ID,
		Email:     user.Email,
		RoleID:    user.RoleID,
		RoleName:  roleName,
		CompanyID: user.CompanyID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			Issuer:    "pkl-backend-api",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", 0, fmt.Errorf("failed to sign access token: %w", err)
	}

	return tokenString, int64(AccessTokenDuration.Seconds()), nil
}

func GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func HashRefreshToken(token string, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(token))
	return hex.EncodeToString(h.Sum(nil))
}

func SetRefreshTokenCookie(c *fiber.Ctx, token string, remember bool, isProd bool) {
	duration := RefreshTokenDuration
	if remember {
		duration = RememberMeDuration
	}

	expiresAt := time.Now().Add(duration)

	c.Cookie(&fiber.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    token,
		Path:     "/api/v1/auth",
		Expires:  expiresAt,
		MaxAge:   int(duration.Seconds()),
		HTTPOnly: true,
		Secure:   isProd,
		SameSite: fiber.CookieSameSiteLaxMode,
	})
}

func ClearRefreshTokenCookie(c *fiber.Ctx, isProd bool) {
	c.Cookie(&fiber.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    "",
		Path:     "/api/v1/auth",
		Expires:  time.Now().Add(-24 * time.Hour),
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   isProd,
		SameSite: fiber.CookieSameSiteLaxMode,
	})
}

func GetRefreshToken(c *fiber.Ctx) string {
	// First check cookie
	if cookie := c.Cookies(RefreshTokenCookieName); cookie != "" {
		return cookie
	}
	// Fallback to X-Refresh-Token header
	if header := c.Get("X-Refresh-Token"); header != "" {
		return header
	}
	return ""
}
