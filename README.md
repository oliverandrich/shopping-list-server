# Shopping List Server

A REST API server for managing shopping lists with passwordless authentication via magic links and comprehensive invitation system.

## Features

- **Passwordless Authentication** - Magic links sent via email with 6-digit codes
- **JWT-based Session Management** - Secure 30-day token expiry
- **Multi-user Support** - Isolated shopping lists with sharing capabilities
- **Invitation System** - Server and list-specific invitations with email notifications
- **Comprehensive Validation** - Input validation with user-friendly error messages
- **Multi-list Support** - Users can create and manage multiple shopping lists
- **Permission System** - Owner/member roles with proper access controls
- **RESTful API** - Complete CRUD operations for lists and items
- **SQLite Database** - Simple deployment with auto-migration

## Quick Start

### 1. Install Dependencies
```bash
go mod download
```

### 2. Set up environment variables:
```bash
export SMTP_HOST=smtp.gmail.com
export SMTP_PORT=587
export SMTP_USER=your-email@gmail.com
export SMTP_PASS=your-app-password
export SMTP_FROM=your-email@gmail.com
export JWT_SECRET=your-secret-key
```

### 3. Initialize the system:
```bash
# Setup admin user and create initial system settings
go run main.go setup
# or using justfile (recommended)
just setup
```

### 4. Run the server:
```bash
# Direct execution
go run main.go
# or using justfile
just run
```

The server will start on port 3000 (or the port specified in the PORT environment variable).

## API Endpoints

### Public Routes
- `GET /api/v1/health` - Health check
- `POST /api/v1/auth/login` - Request magic link (requires valid email)
- `POST /api/v1/auth/verify` - Verify magic link and get JWT

### Protected Routes
All protected routes require a JWT token in the Authorization header: `Bearer <token>`

#### Lists
- `GET /api/v1/lists` - Get all user's lists
- `POST /api/v1/lists` - Create new list
- `GET /api/v1/lists/:id` - Get list details
- `PUT /api/v1/lists/:id` - Update list (owner only)
- `DELETE /api/v1/lists/:id` - Delete list (owner only)
- `GET /api/v1/lists/:id/members` - Get list members
- `DELETE /api/v1/lists/:id/members/:userId` - Remove member

#### List Items
- `GET /api/v1/lists/:id/items` - Get items in list
- `POST /api/v1/lists/:id/items` - Create item in list
- `PUT /api/v1/lists/:id/items/:itemId` - Update item
- `POST /api/v1/lists/:id/items/:itemId/toggle` - Toggle completion
- `DELETE /api/v1/lists/:id/items/:itemId` - Delete item

#### Invitations
- `POST /api/v1/invitations` - Create invitation (server or list)
- `GET /api/v1/invitations` - Get sent invitations
- `DELETE /api/v1/invitations/:id` - Revoke invitation

## Project Structure

```
shopping-list-server/
├── main.go                    # Entry point, CLI commands, server setup
├── justfile                   # Task automation (recommended)
└── internal/
    ├── models/               # Data models and DTOs
    ├── handlers/             # HTTP request handlers
    ├── auth/                 # Authentication logic
    ├── lists/                # Shopping list operations
    ├── invitations/          # Invitation system
    ├── validation/           # Request validation
    ├── setup/                # System setup and migration
    ├── db/                   # Database initialization
    ├── config/               # Configuration management
    └── testutils/            # Test utilities
```

## Development

### Using Just (Recommended)
```bash
# Show all available commands
just

# Project setup and maintenance
just deps          # Install dependencies
just setup          # Initial setup (creates admin user)
just build          # Build the application
just run            # Run the server
just dev            # Development server with auto-restart

# Code quality
just fmt            # Format Go code
just vet            # Vet code for issues  
just check          # Run fmt, vet, and build
just test           # Run tests
just maintenance    # Full maintenance cycle

# Utilities
just clean          # Clean build artifacts
just info           # Show project information
```

### Direct Go Commands
```bash
# Download dependencies
go mod download

# Run tests
go test ./...

# Build binary
go build -o shopping-list-server

# Format code
go fmt ./...

# Vet code for issues
go vet ./...
```

## Environment Variables

The application requires these environment variables:

- `SMTP_HOST` - SMTP server host (defaults to smtp.gmail.com)
- `SMTP_PORT` - SMTP server port (defaults to 587)  
- `SMTP_USER` - SMTP username for sending emails
- `SMTP_PASS` - SMTP password
- `SMTP_FROM` - Sender email address
- `JWT_SECRET` - Secret key for JWT tokens (use a strong secret in production)

## System Setup

### Initial Setup
1. Run `./shopping-list-server setup` or `just setup`
2. Enter admin email address
3. System creates initial admin user and default shopping list
4. Server can now be started

### Authentication Flow  
1. User requests magic link with email (validated format required)
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

## Validation

All API endpoints include comprehensive input validation:

- **Email Format Validation** - Ensures proper email format
- **Required Field Validation** - Prevents empty required fields
- **User-Friendly Error Messages** - Clear, actionable error descriptions

Example validation error response:
```json
{
  "error": "Validation failed",
  "details": {
    "email": "Must be a valid email address"
  }
}
```

## License

EUPL-1.2