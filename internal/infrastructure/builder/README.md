# SnapDeploy Builder Service

The Builder Service is responsible for building, tagging, and pushing Docker images for deployments. It integrates with the deployment system to provide real-time logging and status updates.

## Architecture

```
builder/
├── builder_service.go       # Main orchestrator
├── docker_client.go         # Docker operations wrapper
├── template_generator.go    # Dockerfile template generation
└── log_manager.go          # Log file management
```

## Features

✅ **Multi-language Support**: Supports NODE, NODE_TS, NEXTJS, GO, and PYTHON projects
✅ **Template-based Dockerfiles**: Automatically generates optimized Dockerfiles
✅ **Real-time Logging**: Streams build logs to both files and database
✅ **Status Tracking**: Updates deployment status through the pipeline
✅ **Docker Integration**: Full Docker build, tag, and push capabilities

## Usage

### 1. Initialize the Builder Service

```go
import (
    "snapdeploy-core/internal/infrastructure/builder"
    "snapdeploy-core/internal/infrastructure/persistence"
)

// Initialize dependencies
deploymentRepo := persistence.NewDeploymentRepository(db)

// Create builder service
builderService, err := builder.NewBuilderService(
    deploymentRepo,
    "/tmp/snapdeploy/builds",  // Working directory
    "/tmp/snapdeploy/logs",    // Log directory
)
if err != nil {
    log.Fatal(err)
}
defer builderService.Close()
```

### 2. Build a Deployment

```go
// Prepare build request
buildReq := builder.BuildRequest{
    Deployment:     deployment,      // Deployment entity
    Project:        project,          // Project entity with commands
    RepositoryPath: "/path/to/repo", // Cloned repository path
    ImageTag:       "registry.example.com/my-app:abc123", // Image tag
}

// Execute build (runs asynchronously in production)
err = builderService.BuildDeployment(ctx, buildReq)
if err != nil {
    log.Printf("Build failed: %v", err)
}
```

### 3. Integration with Deployment Handler

Here's how to integrate the builder with your deployment creation endpoint:

```go
// In your deployment handler
func (h *DeploymentHandler) CreateDeployment(c *gin.Context) {
    // ... create deployment entity ...

    // Save initial deployment (status: PENDING)
    response, err := h.deploymentService.CreateDeployment(c.Request.Context(), userID, &req)
    if err != nil {
        // handle error
    }

    // Return immediately to client
    c.JSON(http.StatusCreated, response)

    // Trigger async build process
    go func() {
        ctx := context.Background()

        // 1. Clone repository
        repoPath, err := cloneRepository(project.RepositoryURL(), commitHash)
        if err != nil {
            // Update deployment status to FAILED
            return
        }
        defer cleanupRepo(repoPath)

        // 2. Build and deploy
        buildReq := builder.BuildRequest{
            Deployment:     deployment,
            Project:        project,
            RepositoryPath: repoPath,
            ImageTag:       fmt.Sprintf("registry.example.com/%s:%s",
                project.ID(), deployment.CommitHash()),
        }

        err = builderService.BuildDeployment(ctx, buildReq)
        if err != nil {
            log.Printf("Build failed: %v", err)
        }

        // 3. Cleanup
        builderService.CleanupBuildArtifacts(repoPath)
    }()
}
```

## Log Management

### Log File Format

Logs are stored in the format: `{deployment_id}_{timestamp}.log`

Example: `550e8400-e29b-41d4-a716-446655440000_20231107_143025.log`

### Log Location

Logs are stored in two places:

1. **File System**: `/tmp/snapdeploy/logs/` (configurable)
2. **Database**: Stored in the `deployments.logs` field

### Retrieving Logs

```go
// Get logs from file
logs, err := builderService.GetDeploymentLogs(deployment)

// Or get from database (includes real-time updates)
deployment, err := deploymentService.GetDeploymentByID(ctx, deploymentID)
logs := deployment.Logs()
```

## Dockerfile Templates

The builder includes optimized multi-stage Dockerfile templates for each language:

### Node.js

- Uses `node:18-alpine` for smaller image size
- Multi-stage build to separate dependencies from runtime
- Caches `node_modules` effectively

### Node.js + TypeScript

- Compiles TypeScript in builder stage
- Only includes compiled JavaScript in production image
- Production dependencies only

### Next.js

- Optimized for Next.js standalone output
- Includes static assets and server components
- Runs as non-root user for security

### Go

- Compiles to static binary in builder stage
- Uses `alpine:latest` for minimal runtime (~5MB)
- Includes CA certificates for HTTPS

### Python

- Uses `python:3.11-slim` for balance of size and compatibility
- Virtual environment support
- Handles both `requirements.txt` and `pyproject.toml`

## Environment Configuration

```bash
# Docker configuration
DOCKER_HOST=unix:///var/run/docker.sock

# Builder configuration
BUILDER_WORK_DIR=/tmp/snapdeploy/builds
BUILDER_LOG_DIR=/tmp/snapdeploy/logs

# Registry configuration (for pushing images)
DOCKER_REGISTRY=registry.example.com
DOCKER_REGISTRY_USERNAME=your-username
DOCKER_REGISTRY_PASSWORD=your-password
```

## Build Pipeline Stages

The builder follows this pipeline:

1. **PENDING** → **BUILDING**

   - Generate Dockerfile from template
   - Write Dockerfile to repository
   - Build Docker image
   - Stream build output to logs

2. **BUILDING** → **DEPLOYING**

   - Tag image with deployment info
   - Push image to registry
   - Stream push output to logs

3. **DEPLOYING** → **DEPLOYED**
   - Verify image is available
   - Update deployment status
   - Cleanup temporary files

If any step fails, status is updated to **FAILED** with error details in logs.

## Error Handling

All errors are:

- Logged to the deployment log file
- Stored in the deployment logs field
- Reflected in the deployment status
- Returned to the caller (if synchronous)

## Future Enhancements

- [ ] Registry authentication for private registries
- [ ] Build caching for faster rebuilds
- [ ] Resource limits (CPU, memory) for builds
- [ ] Parallel builds for multiple deployments
- [ ] Build queue management
- [ ] Webhook notifications on build completion
- [ ] Build metrics and monitoring

## Example: Complete Deployment Flow

```go
// 1. User creates deployment via API
POST /api/v1/deployments
{
    "project_id": "uuid",
    "commit_hash": "abc123",
    "branch": "main"
}

// 2. Handler creates deployment entity and returns immediately
// Status: PENDING

// 3. Async worker starts build process
// Status: BUILDING
// Logs: "Starting build process..."
// Logs: "Generating Dockerfile for NODE project..."
// Logs: "Building Docker image..."

// 4. Build completes, starts push
// Status: DEPLOYING
// Logs: "Pushing image to registry..."

// 5. Push completes successfully
// Status: DEPLOYED
// Logs: "Deployment completed successfully!"

// 6. Client polls GET /api/v1/deployments/:id to track progress
// Or subscribes to WebSocket for real-time updates (future feature)
```

## Testing

```bash
# Ensure Docker is running
docker ps

# Run builder tests
go test ./internal/infrastructure/builder/...

# Test with a sample project
make test-build
```

## Security Considerations

1. **Docker Socket Access**: Builder needs access to Docker socket - ensure proper permissions
2. **Registry Credentials**: Store in secure vault (e.g., AWS Secrets Manager)
3. **Resource Limits**: Set memory/CPU limits to prevent DoS
4. **Image Scanning**: Integrate vulnerability scanning before deployment
5. **Cleanup**: Automatically remove old images to prevent disk exhaustion

