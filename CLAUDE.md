# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based REST API server for a shopping list application with passwordless authentication via magic links sent by email.

## Technology Stack

- **Language**: Go 1.24.5
- **Web Framework**: Fiber v2
- **Database**: SQLite with GORM ORM
- **Authentication**: JWT tokens with magic link email authentication
- **Email**: gomail.v2 for SMTP

## Common Development Commands

The project includes a `justfile` for easy task management. Run `just` to see all available commands.

### Using Just (Recommended)
```bash
# Show all available commands
just

# Project setup and maintenance
just deps          # Install dependencies
just setup          # Initial setup (creates admin user)
just build          # Build the application
just run            # Run the server
just dev            # Development server with auto-restart (requires air)

# Code quality
just fmt            # Format Go code
just vet            # Vet code for issues  
just check          # Run fmt, vet, and build
just test           # Run tests
just maintenance    # Full maintenance cycle

# Utilities
just clean          # Clean build artifacts
just info           # Show project information
just audit          # Security audit (requires govulncheck)
just update         # Update dependencies
```

### Direct Go Commands
```bash
# Download dependencies
go mod download

# Initial setup (required before first use)
go run main.go setup
# or
./shopping-list-server setup

# Run the server
go run main.go
# or 
./shopping-list-server

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
├── main.go                    # Entry point, CLI commands, server setup
└── internal/
    ├── models/               # Data models and DTOs
    │   └── models.go
    ├── handlers/             # HTTP request handlers
    │   └── handlers.go
    ├── auth/                 # Authentication logic
    │   └── auth.go
    ├── lists/                # Shopping list operations
    │   └── lists.go
    ├── invitations/          # Invitation system
    │   └── invitations.go
    ├── setup/                # System setup and migration
    │   └── setup.go
    ├── db/                   # Database initialization
    │   └── db.go
    └── config/               # Configuration management
        └── config.go
```

### Package Overview

- **models** - All data structures (User, ShoppingList, ListMember, Invitation, etc.)
- **handlers** - HTTP handlers for all endpoints with permission checks
- **auth** - JWT generation/validation, magic link logic with invitation support
- **lists** - Shopping list CRUD operations and permission management
- **invitations** - Invitation creation, email sending, and acceptance
- **setup** - System initialization, admin creation, data migration
- **db** - Database connection and auto-migration
- **config** - Environment variable management
- **main** - CLI commands, server initialization and route setup

### API Endpoints

**Public Routes:**
- `GET /api/v1/health` - Health check
- `POST /api/v1/auth/login` - Request magic link (email required)
- `POST /api/v1/auth/verify` - Verify magic link and accept invitations

**Protected Routes (require JWT Bearer token):**

**Lists:**
- `GET /api/v1/lists` - Get all user's lists
- `POST /api/v1/lists` - Create new list
- `GET /api/v1/lists/:id` - Get list details
- `PUT /api/v1/lists/:id` - Update list (owner only)
- `DELETE /api/v1/lists/:id` - Delete list (owner only)
- `GET /api/v1/lists/:id/members` - Get list members
- `DELETE /api/v1/lists/:id/members/:userId` - Remove member

**List Items:**
- `GET /api/v1/lists/:id/items` - Get items in list
- `POST /api/v1/lists/:id/items` - Create item in list
- `PUT /api/v1/lists/:id/items/:itemId` - Update item
- `POST /api/v1/lists/:id/items/:itemId/toggle` - Toggle completion
- `DELETE /api/v1/lists/:id/items/:itemId` - Delete item

**Invitations:**
- `POST /api/v1/invitations` - Create invitation (server or list)
- `GET /api/v1/invitations` - Get sent invitations
- `DELETE /api/v1/invitations/:id` - Revoke invitation

## Database

- SQLite database file: `shopping.db`
- Auto-migration runs on startup
- Uses GORM for all database operations

## Setup and Invitation Flow

### Initial Setup
1. Run `./shopping-list-server setup` 
2. Enter admin email address
3. System creates initial admin user and default shopping list
4. Server can now be started

### Authentication Flow  
1. User requests magic link with email
2. Server generates 6-digit code and sends email
3. User verifies code within 15 minutes
4. If user has pending invitation, it's automatically accepted
5. Server returns JWT token (30-day expiry)
6. Client includes token in Authorization header for protected routes

### Invitation System
- **Server Invitations**: Allow new users to join the system
- **List Invitations**: Allow existing users to join specific lists  
- Invitations expire after 7 days
- Invitations are automatically accepted during magic link verification
- Only list owners can invite users to their lists
- New users with server invitations get a default list created

## Development Notes

- **Invitation-only**: New users must have a valid invitation to register
- Server runs on port 3000 (configurable via PORT env var)
- CORS is enabled for all origins
- No test files currently exist
- Code is organized into internal packages for maintainability
- Uses standard Go project layout with internal packages
- Comprehensive permission system (owner/member roles)
- Automatic data migration for existing installations

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