package middleware

import (
	"errors"
	"strings"

	"pkl-golang-be/internal/config"
	"pkl-golang-be/internal/shared/apperror"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const AuthUserKey = "auth_user"

type AuthClaims struct {
	UserID    uuid.UUID  `json:"uid"`
	Email     string     `json:"email"`
	RoleID    uuid.UUID  `json:"role_id"`
	RoleName  string     `json:"role_name"`
	CompanyID *uuid.UUID `json:"company_id,omitempty"`
	jwt.RegisteredClaims
}

func NewAuthMiddleware(cfg *config.Config) fiber.Handler {
	secretBytes := []byte(cfg.JWTSecret)

	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return apperror.Unauthorized("Header Authorization diperlukan")
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return apperror.Unauthorized("Format header Authorization harus 'Bearer <token>'")
		}

		tokenString := parts[1]
		claims := &AuthClaims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return secretBytes, nil
		})

		if err != nil || !token.Valid {
			return apperror.Unauthorized("Token tidak valid atau telah kedaluwarsa")
		}

		c.Locals(AuthUserKey, claims)
		return c.Next()
	}
}

func GetAuthUser(c *fiber.Ctx) (*AuthClaims, bool) {
	claims, ok := c.Locals(AuthUserKey).(*AuthClaims)
	if !ok || claims == nil {
		return nil, false
	}
	return claims, true
}
