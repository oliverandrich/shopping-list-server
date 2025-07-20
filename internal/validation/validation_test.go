// Licensed under the EUPL-1.2-or-later
// Copyright (C) 2025 Oliver Andrich

package validation

import (
	"testing"

	"github.com/oliverandrich/shopping-list-server/internal/models"
)

func TestValidateStruct(t *testing.T) {
	t.Run("valid email in LoginRequest", func(t *testing.T) {
		req := models.LoginRequest{
			Email: "test@example.com",
		}

		err := ValidateStruct(req)
		if err != nil {
			t.Errorf("Expected no validation error, got: %v", err)
		}
	})

	t.Run("invalid email in LoginRequest", func(t *testing.T) {
		req := models.LoginRequest{
			Email: "invalid-email",
		}

		err := ValidateStruct(req)
		if err == nil {
			t.Error("Expected validation error for invalid email")
		}
	})

	t.Run("empty email in LoginRequest", func(t *testing.T) {
		req := models.LoginRequest{
			Email: "",
		}

		err := ValidateStruct(req)
		if err == nil {
			t.Error("Expected validation error for empty email")
		}
	})

	t.Run("valid VerifyRequest", func(t *testing.T) {
		req := models.VerifyRequest{
			Email: "test@example.com",
			Code:  "123456",
		}

		err := ValidateStruct(req)
		if err != nil {
			t.Errorf("Expected no validation error, got: %v", err)
		}
	})

	t.Run("invalid VerifyRequest - missing code", func(t *testing.T) {
		req := models.VerifyRequest{
			Email: "test@example.com",
			Code:  "",
		}

		err := ValidateStruct(req)
		if err == nil {
			t.Error("Expected validation error for missing code")
		}
	})
}

func TestFormatValidationErrors(t *testing.T) {
	t.Run("format email validation error", func(t *testing.T) {
		req := models.LoginRequest{
			Email: "invalid-email",
		}

		err := ValidateStruct(req)
		if err == nil {
			t.Fatal("Expected validation error")
		}

		errors := FormatValidationErrors(err)

		if len(errors) == 0 {
			t.Error("Expected formatted errors")
		}

		if emailError, exists := errors["email"]; !exists {
			t.Error("Expected email validation error")
		} else if emailError != "Must be a valid email address" {
			t.Errorf("Expected 'Must be a valid email address', got: %s", emailError)
		}
	})

	t.Run("format required field error", func(t *testing.T) {
		req := models.LoginRequest{
			Email: "",
		}

		err := ValidateStruct(req)
		if err == nil {
			t.Fatal("Expected validation error")
		}

		errors := FormatValidationErrors(err)

		if emailError, exists := errors["email"]; !exists {
			t.Error("Expected email validation error")
		} else if emailError != "This field is required" {
			t.Errorf("Expected 'This field is required', got: %s", emailError)
		}
	})
}

func TestGetErrorMessage(t *testing.T) {
	// This function is not exported, but we can test it indirectly through FormatValidationErrors
	testCases := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "required field",
			input:    models.LoginRequest{Email: ""},
			expected: "This field is required",
		},
		{
			name:     "invalid email",
			input:    models.LoginRequest{Email: "invalid"},
			expected: "Must be a valid email address",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateStruct(tc.input)
			if err == nil {
				t.Fatal("Expected validation error")
			}

			errors := FormatValidationErrors(err)
			if emailError, exists := errors["email"]; !exists {
				t.Error("Expected email validation error")
			} else if emailError != tc.expected {
				t.Errorf("Expected '%s', got: %s", tc.expected, emailError)
			}
		})
	}
}
