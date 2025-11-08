# PORT Environment Variable Handling

## 🔌 Overview

The `PORT` environment variable is handled specially in SnapDeploy to ensure proper networking configuration throughout the build and deployment pipeline.

## 🏗️ How PORT Works

### Default Behavior

**Default PORT**: `8080`

All containers default to listening on port `8080` unless explicitly overridden via environment variables.

### PORT Flow

```
┌─────────────────────────────────────────────────────────────┐
│               Build Time (Dockerfile)                       │
│  - ENV PORT=8080 (set in Dockerfile)                        │
│  - EXPOSE 8080 (Docker port exposure)                       │
│  - Application reads $PORT to know which port to listen on  │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│            Runtime (ECS Container)                          │
│  - PORT=8080 (injected as environment variable)             │
│  - User can override: PORT=3000 via project env vars        │
│  - Container listens on the specified port                  │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│          ALB Target Group                                   │
│  - Health check on port 8080 (or custom)                    │
│  - Traffic forwarded to container on correct port           │
└─────────────────────────────────────────────────────────────┘
```

## 📝 Dockerfile Templates

All Dockerfile templates now include PORT:

### Node.js Example

```dockerfile
FROM node:18-alpine

# Set PORT environment variable
ENV PORT=8080

# Expose port
EXPOSE 8080

# Run application
CMD npm start
```

**In your app** (`server.js`):

```javascript
const PORT = process.env.PORT || 8080;
app.listen(PORT, () => {
  console.log(`Server running on port ${PORT}`);
});
```

### Python Example

```dockerfile
FROM python:3.11-slim

# Set PORT environment variable
ENV PORT=8080

# Expose port
EXPOSE 8080

# Run application
CMD python main.py
```

**In your app** (`main.py`):

```python
import os
PORT = int(os.getenv('PORT', 8080))
app.run(host='0.0.0.0', port=PORT)
```

### Go Example

```dockerfile
FROM alpine:latest

# Set PORT environment variable
ENV PORT=8080

# Expose port
EXPOSE 8080

# Run application
CMD ./app
```

**In your app** (`main.go`):

```go
port := os.Getenv("PORT")
if port == "" {
    port = "8080"
}
http.ListenAndServe(":"+port, handler)
```

## 🎛️ Customizing PORT

Users can override the default PORT via environment variables:

### Option 1: Set via API

```bash
POST /api/v1/projects/{id}/env
{
  "key": "PORT",
  "value": "3000"
}
```

### Option 2: Set via UI

In the project settings, add environment variable:

- **Key**: `PORT`
- **Value**: `3000`

**Result**:

- Container will listen on port `3000`
- ALB target group will route to port `3000`
- Health checks will use port `3000`

## 🚀 Deployment with Custom PORT

### Example

```bash
# 1. Create project
POST /api/v1/users/{id}/projects
{
  "repository_url": "https://github.com/user/my-app",
  "language": "NODE",
  ...
}

# 2. Set custom PORT
POST /api/v1/projects/{project_id}/env
{
  "key": "PORT",
  "value": "3000"
}

# 3. Deploy
POST /api/v1/deployments
{
  "project_id": "{project_id}",
  ...
}
```

### Deployment Logs

```
🔐 Loading environment variables...
✅ Loaded 1 custom environment variables
🔌 Using custom PORT: 3000                    ← Custom port detected!
🔧 Creating ALB target group and routing rule...
  → Target group port: 3000
  → Health check port: 3000
✅ ALB routing configured
📦 Deploying service with PORT=3000
✅ ECS service created
```

### In Container

```bash
$ echo $PORT
3000

$ curl localhost:3000/health
OK
```

## 🎯 Port Validation

**Valid ports**: `1-65535`

**Common ports**:

- `8080` - Default (HTTP alternative)
- `3000` - Common for Node/React dev
- `8000` - Common for Python/Django
- `4000` - Common for GraphQL
- `5000` - Common for Flask
- `8888` - Common for Jupyter

**Reserved/privileged ports** (`1-1023`):

- Containers run as non-root, so ports < 1024 won't bind
- Use ports >= 1024 (like 8080, 3000, etc.)

## 🔒 Security

The PORT environment variable:

- ✅ Can be set like any other env var
- ✅ Value is encrypted at rest
- ✅ Shown masked in API: `3*******0` or `8*******0`
- ✅ Never exposed to frontend
- ✅ Decrypted only during deployment

## ⚙️ Advanced: Multi-Port Applications

If your application needs multiple ports:

```bash
# Main app port (for ALB routing)
PORT=8080

# Additional ports (exposed but not routed via ALB)
METRICS_PORT=9090
ADMIN_PORT=8081
```

**Note**: ALB only routes to the primary `PORT`. Additional ports are available within the container but not externally accessible through the ALB.

## 🧪 Testing

### Test 1: Default PORT (8080)

```bash
# Deploy without setting PORT env var
# Should use 8080

# Verify
curl https://my-app.snap-deploy.com/
# Works on port 8080
```

### Test 2: Custom PORT (3000)

```bash
# Set PORT=3000 via env vars
POST /api/v1/projects/{id}/env
{"key": "PORT", "value": "3000"}

# Deploy
POST /api/v1/deployments {...}

# Verify logs show: "🔌 Using custom PORT: 3000"

# Access
curl https://my-app.snap-deploy.com/
# Works on port 3000 (ALB routes correctly)
```

## 📊 Summary

| Component            | PORT Handling                             |
| -------------------- | ----------------------------------------- |
| **Dockerfile**       | `ENV PORT=8080` + `EXPOSE 8080`           |
| **Build**            | Template uses {{.Port}} (always 8080)     |
| **Runtime**          | Can override via env var: `PORT=3000`     |
| **ALB Target Group** | Uses actual PORT (default 8080 or custom) |
| **ECS Container**    | Receives PORT as env var                  |
| **Health Checks**    | Checks the correct PORT                   |

**Key Feature**: Users can customize PORT without changing Dockerfiles or configuration - just set the `PORT` environment variable!

## ✅ Best Practices

1. **Read PORT in your application**:

   ```javascript
   const PORT = process.env.PORT || 8080;
   ```

2. **Bind to 0.0.0.0** (not localhost):

   ```javascript
   app.listen(PORT, "0.0.0.0");
   ```

3. **Use standard ports** (3000, 8000, 8080, etc.)

4. **Test locally** with same PORT value

5. **Document** in your README what PORT your app expects
