package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Save original environment
	originalEnv := map[string]string{
		"SMTP_HOST": os.Getenv("SMTP_HOST"),
		"SMTP_PORT": os.Getenv("SMTP_PORT"),
		"SMTP_USER": os.Getenv("SMTP_USER"),
		"SMTP_PASS": os.Getenv("SMTP_PASS"),
		"SMTP_FROM": os.Getenv("SMTP_FROM"),
		"JWT_SECRET": os.Getenv("JWT_SECRET"),
		"PORT": os.Getenv("PORT"),
		"DB_PATH": os.Getenv("DB_PATH"),
	}
	
	// Clean up environment
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	t.Run("with environment variables", func(t *testing.T) {
		// Set test environment variables
		os.Setenv("SMTP_HOST", "test.smtp.com")
		os.Setenv("SMTP_PORT", "2525")
		os.Setenv("SMTP_USER", "testuser")
		os.Setenv("SMTP_PASS", "testpass")
		os.Setenv("SMTP_FROM", "test@example.com")
		os.Setenv("JWT_SECRET", "test-secret")
		os.Setenv("PORT", ":8080")
		os.Setenv("DB_PATH", "test.db")

		cfg := Load()

		if cfg.SMTPHost != "test.smtp.com" {
			t.Errorf("Expected SMTPHost to be 'test.smtp.com', got '%s'", cfg.SMTPHost)
		}
		if cfg.SMTPPort != 2525 {
			t.Errorf("Expected SMTPPort to be 2525, got %d", cfg.SMTPPort)
		}
		if cfg.SMTPUser != "testuser" {
			t.Errorf("Expected SMTPUser to be 'testuser', got '%s'", cfg.SMTPUser)
		}
		if cfg.SMTPPass != "testpass" {
			t.Errorf("Expected SMTPPass to be 'testpass', got '%s'", cfg.SMTPPass)
		}
		if cfg.SMTPFrom != "test@example.com" {
			t.Errorf("Expected SMTPFrom to be 'test@example.com', got '%s'", cfg.SMTPFrom)
		}
		if string(cfg.JWTSecret) != "test-secret" {
			t.Errorf("Expected JWTSecret to be 'test-secret', got '%s'", string(cfg.JWTSecret))
		}
		if cfg.ServerPort != ":8080" {
			t.Errorf("Expected ServerPort to be ':8080', got '%s'", cfg.ServerPort)
		}
		if cfg.DBPath != "test.db" {
			t.Errorf("Expected DBPath to be 'test.db', got '%s'", cfg.DBPath)
		}
	})

	t.Run("with default values", func(t *testing.T) {
		// Clear environment variables
		for key := range originalEnv {
			os.Unsetenv(key)
		}

		cfg := Load()

		if cfg.SMTPHost != "smtp.gmail.com" {
			t.Errorf("Expected default SMTPHost to be 'smtp.gmail.com', got '%s'", cfg.SMTPHost)
		}
		if cfg.SMTPPort != 587 {
			t.Errorf("Expected default SMTPPort to be 587, got %d", cfg.SMTPPort)
		}
		if cfg.ServerPort != ":3000" {
			t.Errorf("Expected default ServerPort to be ':3000', got '%s'", cfg.ServerPort)
		}
		if cfg.DBPath != "shopping.db" {
			t.Errorf("Expected default DBPath to be 'shopping.db', got '%s'", cfg.DBPath)
		}
		if len(cfg.JWTSecret) == 0 {
			t.Error("JWTSecret should not be empty even with default")
		}
	})

	t.Run("with invalid SMTP_PORT", func(t *testing.T) {
		os.Setenv("SMTP_PORT", "invalid")
		
		cfg := Load()
		
		// Should fall back to default
		if cfg.SMTPPort != 587 {
			t.Errorf("Expected SMTPPort to fallback to 587 with invalid value, got %d", cfg.SMTPPort)
		}
	})
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "environment variable exists",
			key:          "TEST_KEY",
			defaultValue: "default",
			envValue:     "env_value",
			expected:     "env_value",
		},
		{
			name:         "environment variable does not exist",
			key:          "NON_EXISTENT_KEY",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up
			defer os.Unsetenv(tt.key)
			
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
			}
			
			result := getEnvOrDefault(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvOrDefault() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetEnvAsIntOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int
		envValue     string
		expected     int
	}{
		{
			name:         "valid integer environment variable",
			key:          "TEST_INT_KEY",
			defaultValue: 100,
			envValue:     "200",
			expected:     200,
		},
		{
			name:         "invalid integer environment variable",
			key:          "TEST_INT_KEY",
			defaultValue: 100,
			envValue:     "invalid",
			expected:     100,
		},
		{
			name:         "environment variable does not exist",
			key:          "NON_EXISTENT_INT_KEY",
			defaultValue: 100,
			envValue:     "",
			expected:     100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up
			defer os.Unsetenv(tt.key)
			
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
			}
			
			result := getEnvAsIntOrDefault(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvAsIntOrDefault() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfigStruct(t *testing.T) {
	cfg := &Config{
		SMTPHost:   "test.smtp.com",
		SMTPPort:   587,
		SMTPUser:   "user",
		SMTPPass:   "pass",
		SMTPFrom:   "from@example.com",
		JWTSecret:  []byte("secret"),
		ServerPort: ":3000",
		DBPath:     "test.db",
	}

	if cfg.SMTPHost == "" {
		t.Error("Config SMTPHost should not be empty")
	}
	if cfg.SMTPPort == 0 {
		t.Error("Config SMTPPort should not be zero")
	}
	if len(cfg.JWTSecret) == 0 {
		t.Error("Config JWTSecret should not be empty")
	}
	if cfg.ServerPort == "" {
		t.Error("Config ServerPort should not be empty")
	}
	if cfg.DBPath == "" {
		t.Error("Config DBPath should not be empty")
	}
}

func TestJWTSecretGeneration(t *testing.T) {
	// Test that JWT secret is generated when not provided
	os.Unsetenv("JWT_SECRET")
	
	cfg1 := Load()
	cfg2 := Load()
	
	if len(cfg1.JWTSecret) == 0 {
		t.Error("JWTSecret should be generated when not provided")
	}
	
	if len(cfg2.JWTSecret) == 0 {
		t.Error("JWTSecret should be generated when not provided")
	}
	
	// Note: Secrets might be the same if generated deterministically
	// This is acceptable for config loading consistency
	t.Logf("Generated JWT secrets - cfg1: %d bytes, cfg2: %d bytes", len(cfg1.JWTSecret), len(cfg2.JWTSecret))
}

func TestPortFormatting(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected string
	}{
		{
			name:     "port with colon",
			envValue: ":8080",
			expected: ":8080",
		},
		{
			name:     "port without colon",
			envValue: "8080",
			expected: "8080", // The config should store exactly what's provided
		},
		{
			name:     "default port",
			envValue: "",
			expected: ":3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer os.Unsetenv("PORT")
			
			if tt.envValue != "" {
				os.Setenv("PORT", tt.envValue)
			}
			
			cfg := Load()
			if cfg.ServerPort != tt.expected {
				t.Errorf("Expected ServerPort to be '%s', got '%s'", tt.expected, cfg.ServerPort)
			}
		})
	}
}