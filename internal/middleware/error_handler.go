package middleware

import (
	"errors"
	"log/slog"
	"net/http"

	"pkl-golang-be/internal/shared/apperror"
	"pkl-golang-be/internal/shared/response"

	"github.com/gofiber/fiber/v2"
)

func CustomErrorHandler(c *fiber.Ctx, err error) error {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		if appErr.Type == apperror.TypeInternalServer {
			slog.Error("Internal server error", "error", appErr.Err, "path", c.Path())
		}
		return response.Error(c, appErr.HTTPStatus(), appErr.Message, appErr.Details)
	}

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		return response.Error(c, fiberErr.Code, fiberErr.Message)
	}

	slog.Error("Unhandled exception", "error", err, "path", c.Path())
	return response.Error(c, http.StatusInternalServerError, "An unexpected internal server error occurred")
}
