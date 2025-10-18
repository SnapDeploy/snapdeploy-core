# Clerk Authentication Setup

This document explains how to set up Clerk authentication for the SnapDeploy application.

## Backend Configuration

### 1. Environment Variables

Create a `.env` file in the `snapdeploy-core` directory with the following variables:

```env
# Server Configuration
SERVER_PORT=8080
SERVER_HOST=0.0.0.0
SERVER_READ_TIMEOUT=30
SERVER_WRITE_TIMEOUT=30
SERVER_IDLE_TIMEOUT=120

# Database Configuration
DB_DRIVER=postgres
DB_DSN=postgres://snapdeploy:snapdeploy123@localhost:5433/snapdeploy?sslmode=disable
DB_MAX_CONNS=25
DB_MIN_CONNS=5

# Clerk Configuration
CLERK_PUBLISHABLE_KEY=pk_test_your-publishable-key
CLERK_SECRET_KEY=sk_test_your-secret-key
CLERK_JWKS_URL=https://your-clerk-instance.clerk.accounts.dev/.well-known/jwks.json
CLERK_ISSUER=https://your-clerk-instance.clerk.accounts.dev
```

### 2. Database Migration

Run the database migration to update the schema:

```bash
# Start your PostgreSQL database
# Then run the migration
goose -dir migrations postgres "postgres://snapdeploy:snapdeploy123@localhost:5433/snapdeploy?sslmode=disable" up
```

### 3. Running the Backend

```bash
go run cmd/server/main.go
```

## Frontend Configuration

### 1. Environment Variables

Create a `.env.local` file in the `snapdeploy-ui` directory:

```env
VITE_CLERK_PUBLISHABLE_KEY=pk_test_your-publishable-key
VITE_API_URL=http://localhost:8080/api/v1
```

### 2. Running the Frontend

```bash
cd snapdeploy-ui
pnpm install
pnpm dev
```

## Clerk Dashboard Setup

1. Go to [Clerk Dashboard](https://dashboard.clerk.com/)
2. Create a new application
3. Copy the publishable key and secret key
4. Configure the JWT template to include the following claims:
   - `sub` (subject) - user ID
   - `email` - user email
   - `username` - username
   - `given_name` - first name
   - `family_name` - last name

## Authentication Flow

1. User signs in through Clerk on the frontend
2. Clerk provides a JWT token
3. Frontend sends the token in the `Authorization: Bearer <token>` header
4. Backend verifies the JWT token using Clerk's public keys
5. Backend extracts user information and creates/updates user in database
6. Backend returns user data to frontend

## API Endpoints

All protected endpoints require the `Authorization: Bearer <token>` header.

- `GET /api/v1/auth/me` - Get current user information
- `GET /api/v1/users` - List users (admin only)
- `GET /api/v1/users/:id` - Get user by ID
- `PUT /api/v1/users/:id` - Update user
- `DELETE /api/v1/users/:id` - Delete user

## Testing

To test the authentication flow:

1. Start both backend and frontend
2. Navigate to the frontend URL
3. Click "Sign In to Continue"
4. Complete the Clerk authentication flow
5. Verify that you can access protected endpoints

## Troubleshooting

### Common Issues

1. **Invalid token errors**: Check that the JWT issuer and JWKS URL are correct
2. **CORS errors**: Ensure the frontend URL is allowed in Clerk's CORS settings
3. **Database errors**: Make sure the migration has been run and the database is accessible

### Debug Mode

Enable debug logging by setting:

```env
GIN_MODE=debug
```

This will show detailed request logs and help debug authentication issues.

