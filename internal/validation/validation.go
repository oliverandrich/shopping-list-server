// Licensed under the EUPL-1.2-or-later
// Copyright (C) 2025 Oliver Andrich

// Package validation provides request validation utilities and error formatting.
package validation

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// ValidateStruct validates a struct using the validator tags
func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}

// FormatValidationErrors converts validator errors to a user-friendly map
func FormatValidationErrors(err error) map[string]string {
	errors := make(map[string]string)

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			field := strings.ToLower(e.Field())
			errors[field] = getErrorMessage(e)
		}
	}

	return errors
}

// getErrorMessage returns a user-friendly error message for a validation error
func getErrorMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Must be a valid email address"
	case "min":
		return "Value is too short"
	case "max":
		return "Value is too long"
	case "uuid":
		return "Must be a valid UUID"
	default:
		return "Invalid value"
	}
}
