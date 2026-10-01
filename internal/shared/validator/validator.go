package validator

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func ValidateStruct(s any) []FieldError {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	var errs []FieldError
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			errs = append(errs, FieldError{
				Field:   toSnakeCase(e.Field()),
				Message: formatMessage(e),
			})
		}
	} else {
		errs = append(errs, FieldError{
			Field:   "request",
			Message: err.Error(),
		})
	}

	return errs
}

func formatMessage(e validator.FieldError) string {
	field := toSnakeCase(e.Field())
	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters/items", field, e.Param())
	case "max":
		return fmt.Sprintf("%s must not exceed %s characters/items", field, e.Param())
	case "uuid":
		return fmt.Sprintf("%s must be a valid UUID", field)
	default:
		return fmt.Sprintf("%s failed validation on '%s'", field, e.Tag())
	}
}

func toSnakeCase(str string) string {
	var result []rune
	for i, r := range str {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}
