# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based REST API server for a shopping list application with passwordless authentication via magic links sent by email.

## Technology Stack

- **Language**: Go 1.24.5
- **Web Framework**: Echo v4
- **Database**: SQLite with GORM ORM
- **Authentication**: JWT tokens with magic link email authentication
- **Email**: gomail.v2 for SMTP

## Common Development Commands

```bash
# Download dependencies
go mod download

# Run the server
go run main.go

# Build the binary
go build -o shopping-list-server

# Tidy dependencies
go mod tidy

# Format code
go fmt ./...

# Vet code for issues
go vet ./...

# Run tests (when they exist)
go test ./...
```

## Environment Variables

The application requires these environment variables:

- `SMTP_HOST` - SMTP server host (defaults to smtp.gmail.com)
- `SMTP_PORT` - SMTP server port (defaults to 587)  
- `SMTP_USER` - SMTP username for sending emails
- `SMTP_PASS` - SMTP password
- `SMTP_FROM` - Sender email address
- `JWT_SECRET` - Secret key for JWT tokens (use a strong secret in production)

## Architecture

The application follows a clean, modular structure organized into packages:

```
shopping-list-server/
├── main.go                    # Entry point, server setup
└── internal/
    ├── models/               # Data models and DTOs
    │   └── models.go
    ├── handlers/             # HTTP request handlers
    │   └── handlers.go
    ├── auth/                 # Authentication logic
    │   └── auth.go
    ├── db/                   # Database initialization
    │   └── db.go
    └── config/               # Configuration management
        └── config.go
```

### Package Overview

- **models** - All data structures (User, MagicLink, ShoppingItem, DTOs)
- **handlers** - HTTP handlers for all endpoints
- **auth** - JWT generation/validation, magic link logic, middleware
- **db** - Database connection and auto-migration
- **config** - Environment variable management
- **main** - Server initialization and route setup

### API Endpoints

**Public Routes:**
- `GET /api/v1/health` - Health check
- `POST /api/v1/auth/login` - Request magic link (email required)
- `POST /api/v1/auth/verify` - Verify magic link code and get JWT token

**Protected Routes (require JWT Bearer token):**
- `GET /api/v1/items` - Get user's shopping items
- `POST /api/v1/items` - Create new item
- `PUT /api/v1/items/:id` - Update item
- `POST /api/v1/items/:id/toggle` - Toggle item completion
- `DELETE /api/v1/items/:id` - Delete item

## Database

- SQLite database file: `shopping.db`
- Auto-migration runs on startup
- Uses GORM for all database operations

## Authentication Flow

1. User requests magic link with email
2. Server generates 6-digit code and sends email
3. User verifies code within 15 minutes
4. Server returns JWT token (30-day expiry)
5. Client includes token in Authorization header for protected routes

## Development Notes

- Server runs on port 3000 (configurable via PORT env var)
- CORS is enabled for all origins
- No test files currently exist
- Code is organized into internal packages for maintainability
- Uses standard Go project layout with internal packages
- Basic error handling - enhance with proper logging for production

## Commit Message Convention

Use conventional commit format with the following structure:

```
type(scope): brief description

- Bullet point describing key change
- Another bullet point for significant addition
- Additional points as needed
```

**Types:**
- `feat`: New feature or functionality
- `fix`: Bug fixes
- `refactor`: Code refactoring without functional changes
- `chore`: Maintenance tasks, dependency updates
- `docs`: Documentation updates
- `test`: Adding or updating tests
- `style`: Code style changes (formatting, etc.)

**Scopes (examples):**
- `planning`: Planning board functionality
- `articles`: Article management
- `tasks`: Task management system
- `api`: API endpoints
- `auth`: Authentication/authorization
- `ui`: User interface changes
- `db`: Database changes/migrations

**Example:**
```
feat(planning): implement timetravel functionality for planning cards

- Add pb_timetravel_planning_card view to move publication dates
- Support both offset-based and absolute date targeting
- Include option to move all article variants simultaneously
- Add PbTimeTravelForm with validation for mutually exclusive inputs
```

**Important:** Do NOT include any references to Claude Code, AI assistance, co-authoring information, or any AI-related attribution in commit messages. Keep commits focused on the technical changes only.