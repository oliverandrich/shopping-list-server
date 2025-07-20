# Shopping List Server - Development Justfile

# Default recipe to display available commands
default:
    @just --list

# Install dependencies
deps:
    go mod download
    go mod tidy

# Build the application
build:
    go build -o shopping-list-server

# Run the application
run:
    go run main.go

# Setup the application (creates admin user)
setup:
    go run main.go setup

# Development server with auto-restart (requires air)
dev:
    @if command -v air >/dev/null 2>&1; then \
        air; \
    else \
        echo "Air not installed. Install with: go install github.com/cosmtrek/air@latest"; \
        echo "Falling back to regular run..."; \
        go run main.go; \
    fi

# Format Go code
fmt:
    go fmt ./...

# Vet Go code for issues
vet:
    go vet ./...

# Run tests (when they exist)
test:
    go test ./...

# Run tests with coverage
test-coverage:
    go test -cover ./...

# Run tests with detailed coverage report
test-coverage-html:
    go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report generated: coverage.html"

# Run tests with coverage and show percentage
test-coverage-func:
    go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out

# Clean build artifacts
clean:
    rm -f shopping-list-server
    rm -f coverage.out coverage.html
    go clean

# Check code quality (format, vet, build)
check: fmt vet build
    @echo "✅ Code quality checks passed"

# Full maintenance: deps, check, test with coverage
maintenance: deps check test-coverage
    @echo "✅ Full maintenance completed"

# Show project information
info:
    @echo "Shopping List Server"
    @echo "==================="
    @echo "Framework: Fiber v2"
    @echo "Database: SQLite with GORM"
    @echo "Authentication: JWT + Magic Links"
    @echo ""
    @go version
    @echo ""
    @echo "Available commands:"
    @just --list

# Security audit of dependencies
audit:
    @if command -v govulncheck >/dev/null 2>&1; then \
        govulncheck ./...; \
    else \
        echo "govulncheck not installed. Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
    fi

# Update dependencies to latest versions
update:
    go get -u ./...
    go mod tidy
    @echo "Dependencies updated. Run 'just check' to verify everything still works."