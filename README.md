# Shopping List Server

A simple REST API server for managing shopping lists with passwordless authentication.

## Features

- Passwordless authentication via magic links sent by email
- JWT-based session management
- Multi-user support with isolated shopping lists
- RESTful API for CRUD operations on shopping items
- SQLite database for simple deployment

## Quick Start

1. Set up environment variables:
   ```bash
   export SMTP_HOST=smtp.gmail.com
   export SMTP_PORT=587
   export SMTP_USER=your-email@gmail.com
   export SMTP_PASS=your-app-password
   export SMTP_FROM=your-email@gmail.com
   export JWT_SECRET=your-secret-key
   ```

2. Run the server:
   ```bash
   go run main.go
   ```

The server will start on port 3000 (or the port specified in the PORT environment variable).

## API Endpoints

### Public Routes
- `GET /api/v1/health` - Health check
- `POST /api/v1/auth/login` - Request magic link
- `POST /api/v1/auth/verify` - Verify magic link and get JWT

### Protected Routes
All protected routes require a JWT token in the Authorization header: `Bearer <token>`

- `GET /api/v1/items` - Get all shopping items
- `POST /api/v1/items` - Create a new item
- `PUT /api/v1/items/:id` - Update an item
- `POST /api/v1/items/:id/toggle` - Toggle item completion
- `DELETE /api/v1/items/:id` - Delete an item

## Project Structure

```
shopping-list-server/
├── main.go                    # Entry point
└── internal/
    ├── auth/                  # Authentication logic
    ├── config/                # Configuration management
    ├── db/                    # Database initialization
    ├── handlers/              # HTTP request handlers
    └── models/                # Data models and DTOs
```

## Development

```bash
# Download dependencies
go mod download

# Run tests
go test ./...

# Build binary
go build -o shopping-list-server
```

## License

MIT