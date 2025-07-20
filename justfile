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

# Clean build artifacts
clean:
    rm -f shopping-list-server
    go clean

# Check code quality (format, vet, build)
check: fmt vet build
    @echo "✅ Code quality checks passed"

# Full maintenance: deps, check, test
maintenance: deps check test
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