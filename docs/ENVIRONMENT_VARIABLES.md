# Environment Variables System

## 🔒 Secure Environment Variable Management

SnapDeploy provides a secure system for managing environment variables for your deployed applications.

### Key Features

✅ **Encrypted Storage** - AES-256-GCM encryption at rest  
✅ **Masked API Responses** - Values never exposed to frontend (`f*******t`)  
✅ **Per-Project Isolation** - Each project has its own env vars  
✅ **Automatic Injection** - Loaded into deployed containers  
✅ **No Click-to-Reveal** - Values stay secure (never sent to frontend)  

## 📊 Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                  Frontend (React)                           │
│  - Shows masked values: "A*******Z"                         │
│  - NEVER receives real values                               │
│  - Can add/update/delete vars                               │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼ HTTPS
┌─────────────────────────────────────────────────────────────┐
│                  Backend API                                │
│  - Validates key format (ENV_VAR_NAME)                      │
│  - Encrypts value with AES-256-GCM                          │
│  - Stores encrypted in PostgreSQL                           │
│  - Returns masked: first_char + "******" + last_char        │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│              PostgreSQL Database                            │
│  project_environment_variables                              │
│  ├── id (UUID)                                              │
│  ├── project_id (UUID) → projects(id)                       │
│  ├── key (TEXT) - e.g., "DATABASE_URL"                      │
│  ├── value (TEXT) - Base64 encrypted                        │
│  └── UNIQUE(project_id, key)                                │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼ Decrypted for deployment only
┌─────────────────────────────────────────────────────────────┐
│               ECS Fargate Container                         │
│  - Receives DECRYPTED env vars                              │
│  - Example: DATABASE_URL=postgres://...                     │
│  - Never exposed externally                                 │
└─────────────────────────────────────────────────────────────┘
```

## 🔐 Security Model

### Encryption

- **Algorithm**: AES-256-GCM (Galois/Counter Mode)
- **Key Size**: 32 bytes (256 bits)
- **Key Storage**: Environment variable `ENCRYPTION_KEY` (base64-encoded)
- **Nonce**: Random, unique per value, stored with ciphertext

### Value Masking

Values are masked when returned via API:

| Original Value | Masked Value | Notes |
|----------------|--------------|-------|
| `secret_password_123` | `s*******3` | First + last char |
| `abc` | `***` | Short values fully masked |
| `my_api_key_xyz` | `m*******z` | Always consistent |

**Important**: The real value is NEVER sent to the frontend, even on "reveal" clicks.

## 📡 API Endpoints

### Get Project Environment Variables

```http
GET /api/v1/projects/{project_id}/env
```

**Response**:
```json
{
  "environment_variables": [
    {
      "id": "uuid",
      "project_id": "uuid",
      "key": "DATABASE_URL",
      "value": "p*******s",  // MASKED - never shows real value
      "created_at": "2025-11-08T10:00:00Z",
      "updated_at": "2025-11-08T10:00:00Z"
    }
  ],
  "count": 1
}
```

### Create/Update Environment Variable

```http
POST /api/v1/projects/{project_id}/env
Content-Type: application/json

{
  "key": "DATABASE_URL",
  "value": "postgres://user:pass@host:5432/db"
}
```

**Response**:
```json
{
  "id": "uuid",
  "project_id": "uuid",
  "key": "DATABASE_URL",
  "value": "p*******b",  // MASKED
  "created_at": "2025-11-08T10:00:00Z",
  "updated_at": "2025-11-08T10:00:00Z"
}
```

**Notes**:
- If key already exists, updates the value
- Value is encrypted before storage
- Returns masked value immediately

### Delete Environment Variable

```http
DELETE /api/v1/projects/{project_id}/env/{key}
```

**Response**: `204 No Content`

## 🚀 Deployment Integration

Environment variables are automatically injected into deployed containers:

```go
// During deployment
1. Load env vars from database (encrypted)
2. Decrypt values server-side
3. Pass to ECS task definition
4. Container starts with env vars available
```

**Deployment Logs**:
```
🔐 Loading environment variables...
✅ Loaded 3 environment variables
📦 Deploying service: snapdeploy-abc12345
✅ ECS service created with environment variables
```

**In Container**:
```bash
# Your app can access env vars normally
$ echo $DATABASE_URL
postgres://user:pass@host:5432/db

$ echo $API_KEY
sk-1234567890abcdef
```

## 🛠️ Setup

### 1. Generate Encryption Key

```bash
# Generate a secure 32-byte key
openssl rand -base64 32
```

**Example output**:
```
k7X9mP2nQ4vY6wZ8bC1dE3fG5hJ7kL9mN0pR2sT4uV6=
```

### 2. Add to Environment

**Local Development** (`.env`):
```bash
ENCRYPTION_KEY=k7X9mP2nQ4vY6wZ8bC1dE3fG5hJ7kL9mN0pR2sT4uV6=
```

**Terraform** (`terraform.tfvars`):
```hcl
encryption_key = "k7X9mP2nQ4vY6wZ8bC1dE3fG5hJ7kL9mN0pR2sT4uV6="
```

### 3. Run Migration

```bash
cd /home/franek/code/SnapDeploy/snapdeploy-core
make migrate-up
```

Creates the `project_environment_variables` table.

## 📝 Environment Variable Key Format

Valid keys must follow Unix environment variable naming rules:

✅ **Valid Keys**:
- `DATABASE_URL`
- `API_KEY`
- `MY_SECRET_123`
- `_PRIVATE_VAR`

❌ **Invalid Keys**:
- `123_VAR` (can't start with number)
- `my-var` (no hyphens)
- `my var` (no spaces)
- `my.var` (no dots)

**Rules**:
- Must start with letter or underscore
- Can contain: letters, numbers, underscores
- Max length: 255 characters
- Case-sensitive

## 🎯 Usage Examples

### Frontend (React)

```typescript
// Get environment variables (returns masked values)
const { data } = useQuery({
  queryKey: ['project-env-vars', projectId],
  queryFn: () => api.getProjectEnvVars(projectId)
});

// Display: "D*******B" (masked)
<div>{envVar.value}</div>

// Add/Update
await api.createOrUpdateEnvVar(projectId, {
  key: 'DATABASE_URL',
  value: 'postgres://...'  // Real value sent once
});

// After save: "p*******s" (masked) returned
```

### Backend (Go)

```go
// Create env var
envVar, err := project.NewEnvironmentVariable(
    projectID,
    "API_KEY",
    "sk-1234567890abcdef",  // Plaintext
)

// Save (automatically encrypted)
envVarRepo.Save(ctx, envVar)

// Get for display (masked)
envVars := envVarService.GetProjectEnvVars(ctx, projectID, userID)
// Returns: "s*******f"

// Get for deployment (decrypted)
decrypted := envVarRepo.DecryptAll(ctx, projectID)
// Returns: map[string]string{"API_KEY": "sk-1234567890abcdef"}
```

## 🔄 Deployment Flow with Env Vars

```
1. User adds env vars via UI
   POST /api/v1/projects/{id}/env
   { "key": "DATABASE_URL", "value": "postgres://..." }

2. Backend encrypts and stores
   Encrypted: "aGVsbG8gd29ybGQ..."
   Stored in: project_environment_variables table

3. User creates deployment
   POST /api/v1/deployments
   { "project_id": "...", "branch": "main" }

4. During deployment:
   🔐 Loading environment variables...
   ✅ Loaded 3 environment variables
   📦 Deploying to ECS with decrypted env vars

5. Container starts with env vars:
   export DATABASE_URL="postgres://..."
   export API_KEY="sk-123..."
   export SECRET_TOKEN="abc..."
```

## 🛡️ Security Best Practices

### DO ✅

- ✅ Use strong, random encryption key (32 bytes)
- ✅ Rotate encryption key periodically
- ✅ Keep `ENCRYPTION_KEY` in secure secret management (AWS Secrets Manager)
- ✅ Use HTTPS for all API communication
- ✅ Audit env var access logs

### DON'T ❌

- ❌ Never log real env var values
- ❌ Never send real values to frontend
- ❌ Never store encryption key in code/git
- ❌ Never share encryption key between environments
- ❌ Never use weak/predictable keys

## 🔄 Key Rotation

To rotate the encryption key:

```bash
# 1. Generate new key
NEW_KEY=$(openssl rand -base64 32)

# 2. Set environment variable
export NEW_ENCRYPTION_KEY=$NEW_KEY

# 3. Re-encrypt all values (TODO: Add migration script)
go run scripts/reencrypt-env-vars.go

# 4. Update ENCRYPTION_KEY in all environments
# 5. Deploy updated configuration
```

## 🧪 Testing

### Test Encryption/Decryption

```go
// Generate test key
key, _ := encryption.GenerateKey()
fmt.Println("Key:", key)

// Create service
service, _ := encryption.NewEncryptionService()

// Encrypt
encrypted, _ := service.Encrypt("my_secret_value")
fmt.Println("Encrypted:", encrypted)

// Decrypt
decrypted, _ := service.Decrypt(encrypted)
fmt.Println("Decrypted:", decrypted)
// Output: "my_secret_value"
```

### Test Masking

```go
// Short value
maskValue("abc", nil)  // "***"

// Normal value
maskValue("my_secret_password", nil)  // "m*******d"

// Empty
maskValue("", nil)  // "********"
```

## 📊 Database Schema

```sql
CREATE TABLE project_environment_variables (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    value TEXT NOT NULL,  -- Encrypted with AES-256-GCM
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(project_id, key)  -- One value per key per project
);

CREATE INDEX idx_env_vars_project_id ON project_environment_variables(project_id);
```

##  Troubleshooting

### "ENCRYPTION_KEY environment variable is required"

**Solution**: Generate and set encryption key:
```bash
openssl rand -base64 32  # Generate key
export ENCRYPTION_KEY="generated_key_here"
```

### "failed to decrypt"

**Causes**:
- Encryption key changed
- Database value corrupted
- Wrong key being used

**Solution**: Re-create the env var with correct key.

### "invalid key format"

**Cause**: Key doesn't follow Unix env var naming rules

**Solution**: Use only letters, numbers, underscores. Start with letter/underscore.

## 🚀 Next Steps

1. ✅ Migration applied
2. ✅ Encryption service initialized  
3. ✅ API endpoints available
4. ✅ Deployed containers receive decrypted values
5. **TODO**: Build frontend UI for managing env vars

Frontend will show:
- List of env vars with masked values
- Add new env var
- Update existing (replaces entirely, no reveal)
- Delete env var
- Visual indicator: "🔒 Secured"

