package response

import (
	"net/http"
	"reflect"
	"time"

	"pkl-golang-be/internal/shared/pagination"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Envelope is the standard top-level JSON response envelope for all API endpoints.
type Envelope struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   any    `json:"error,omitempty"`
	Meta    Meta   `json:"meta"`
}

// Meta holds request-level metadata and optional pagination info.
type Meta struct {
	RequestID  string           `json:"request_id"`
	Timestamp  string           `json:"timestamp"`
	Pagination *pagination.Meta `json:"pagination,omitempty"`
}

func newMeta(c *fiber.Ctx, pag *pagination.Meta) Meta {
	reqID := ""
	if id, ok := c.Locals("requestid").(string); ok && id != "" {
		reqID = id
	} else if id := c.Get("X-Request-ID"); id != "" {
		reqID = id
	} else {
		reqID = uuid.NewString()
	}

	return Meta{
		RequestID:  reqID,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Pagination: pag,
	}
}

// Single returns a successful response for a single data object.
func Single(c *fiber.Ctx, data any, message ...string) error {
	msg := "Success"
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}

	return c.Status(http.StatusOK).JSON(Envelope{
		Success: true,
		Message: msg,
		Data:    data,
		Meta:    newMeta(c, nil),
	})
}

// Created returns a 201 Created response for a newly created single resource.
func Created(c *fiber.Ctx, data any, message ...string) error {
	msg := "Resource created successfully"
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}

	return c.Status(http.StatusCreated).JSON(Envelope{
		Success: true,
		Message: msg,
		Data:    data,
		Meta:    newMeta(c, nil),
	})
}

// Paginated returns a standardized list response with mandatory pagination metadata.
// Even if items is empty or nil, Data is guaranteed to serialize as "[]" instead of "null".
func Paginated(c *fiber.Ctx, items any, meta pagination.Meta, message ...string) error {
	msg := "Data retrieved successfully"
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}

	safeItems := ensureSlice(items)

	return c.Status(http.StatusOK).JSON(Envelope{
		Success: true,
		Message: msg,
		Data:    safeItems,
		Meta:    newMeta(c, &meta),
	})
}

// Empty returns a 200 OK response with no data payload (for Delete, Logout, etc.).
func Empty(c *fiber.Ctx, message ...string) error {
	msg := "Operation completed successfully"
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}

	return c.Status(http.StatusOK).JSON(Envelope{
		Success: true,
		Message: msg,
		Meta:    newMeta(c, nil),
	})
}

// Error returns a standardized error response.
func Error(c *fiber.Ctx, statusCode int, message string, details ...any) error {
	var errDetail any
	if len(details) > 0 && details[0] != nil {
		errDetail = details[0]
	}

	return c.Status(statusCode).JSON(Envelope{
		Success: false,
		Message: message,
		Error:   errDetail,
		Meta:    newMeta(c, nil),
	})
}

// ensureSlice guarantees that a nil or empty slice serializes to "[]" instead of "null" in JSON.
func ensureSlice(items any) any {
	if items == nil {
		return []any{}
	}
	v := reflect.ValueOf(items)
	if v.Kind() == reflect.Slice && v.IsNil() {
		return []any{}
	}
	return items
}
