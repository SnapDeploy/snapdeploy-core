# SnapDeploy Core

A modern user management system with AWS Cognito authentication, built with Go, Gin, SQLite, and clean architecture principles.

## Features

- **Health Check**: Basic health monitoring endpoint
- **User Management**: Complete user CRUD operations with AWS Cognito integration
- **Authentication**: JWT-based authentication using AWS Cognito
- **Database**: PostgreSQL with SQLC for type-safe database queries
- **Migrations**: Database schema management with Goose
- **API Documentation**: Auto-generated Swagger/OpenAPI documentation
- **Clean Architecture**: Well-structured codebase with separation of concerns

## Tech Stack

- **Language**: Go 1.24
- **Web Framework**: Gin
- **Database**: PostgreSQL
- **ORM**: SQLC (code generation)
- **Migrations**: Goose
- **Authentication**: AWS Cognito
- **Documentation**: Swagger/OpenAPI
- **Architecture**: Clean Architecture (Handlers → Services → Repositories)

## Project Structure

```
snapdeploy-core/
├── cmd/server/           # Application entry point
├── internal/
│   ├── config/          # Configuration management
│   ├── database/        # Database connection and generated code
│   ├── handlers/        # HTTP handlers
│   ├── middleware/      # HTTP middleware (auth, CORS)
│   ├── models/          # Domain models
│   ├── repositories/    # Data access layer
│   └── services/        # Business logic layer
├── migrations/          # Database migrations
├── sqlc/               # SQL queries for code generation
├── api/                # OpenAPI specifications
├── docs/               # Generated Swagger documentation
└── data/               # SQLite database file
```

## Getting Started

### Prerequisites

- Go 1.24 or later
- Docker and Docker Compose
- AWS Cognito User Pool (for authentication)

### Installation

1. Clone the repository:

```bash
git clone <repository-url>
cd snapdeploy-core
```

2. Install development tools:

```bash
make install-tools
```

3. Install dependencies:

```bash
make deps
```

4. Start PostgreSQL with Docker Compose:

```bash
make docker-up
```

5. Set up environment variables:

```bash
cp env.example .env
# Edit .env with your AWS Cognito configuration
```

6. Run database migrations:

```bash
make migrate-up
```

7. Generate code (SQLC, Swagger):

```bash
make generate
```

8. Build the application:

```bash
make build
```

9. Run the server:

```bash
make run
```

### Configuration

Create a `.env` file with the following variables:

```env
# Server Configuration
SERVER_PORT=8080
SERVER_HOST=0.0.0.0

# Database Configuration
DB_DRIVER=postgres
DB_DSN=postgres://snapdeploy:snapdeploy123@localhost:5433/snapdeploy?sslmode=disable

# AWS Configuration
AWS_REGION=us-east-1
AWS_COGNITO_USER_POOL=your-user-pool-id
AWS_COGNITO_CLIENT_ID=your-client-id

# JWT Configuration
JWT_ISSUER=https://cognito-idp.us-east-1.amazonaws.com/your-user-pool-id
```

## API Endpoints

### Health Check

- `GET /health` - Health check endpoint

### Authentication

- `GET /api/v1/auth/me` - Get current user information (requires authentication)

### User Management

- `GET /api/v1/users` - List users (requires authentication)
- `GET /api/v1/users/:id` - Get user by ID (requires authentication)
- `PUT /api/v1/users/:id` - Update user (requires authentication)
- `DELETE /api/v1/users/:id` - Delete user (requires authentication)

### Documentation

- `GET /swagger/index.html` - Swagger UI documentation

## Development

### Available Make Commands

```bash
make help              # Show available commands
make install-tools     # Install development tools
make setup            # Complete development setup
make build            # Build the application
make run              # Run the application
make test             # Run tests
make clean            # Clean build artifacts
make generate         # Generate code (SQLC, Swagger)
make migrate-up       # Run database migrations up
make migrate-down     # Run database migrations down
make migrate-create   # Create a new migration
make swagger          # Generate Swagger documentation
make sqlc             # Generate SQLC code
make deps             # Download and tidy dependencies
make fmt              # Format code
make lint             # Lint code
make docker-up        # Start PostgreSQL with Docker Compose
make docker-down      # Stop Docker Compose services
make docker-build     # Build Docker image
```

### Adding New Features

1. **Database Changes**: Create a new migration with `make migrate-create`
2. **SQL Queries**: Add queries to `sqlc/queries/` and run `make sqlc`
3. **API Endpoints**: Add handlers in `internal/handlers/`
4. **Business Logic**: Add services in `internal/services/`
5. **Data Access**: Add repositories in `internal/repositories/`

### Code Generation

The project uses several code generation tools:

- **SQLC**: Generates type-safe database code from SQL queries
- **Swagger**: Generates API documentation from code annotations

Run `make generate` to regenerate all code.

## Authentication

The API uses AWS Cognito for authentication. Include the JWT token in the Authorization header:

```
Authorization: Bearer <your-jwt-token>
```

## Database

The application uses PostgreSQL for production-ready data storage. The database runs in a Docker container for easy development setup.

### Migrations

Database schema changes are managed through migrations:

```bash
# Create a new migration
make migrate-create

# Apply migrations
make migrate-up

# Rollback migrations
make migrate-down
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Run `make fmt` and `make lint`
6. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
