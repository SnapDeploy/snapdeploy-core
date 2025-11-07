# Dockerfile Templates

This directory contains Dockerfile templates for different project types. These templates are used by the Builder Service to generate optimized Docker images for deployments.

## Available Templates

- **`node.Dockerfile.tmpl`** - Node.js projects
- **`node_ts.Dockerfile.tmpl`** - Node.js with TypeScript projects
- **`nextjs.Dockerfile.tmpl`** - Next.js applications
- **`go.Dockerfile.tmpl`** - Go applications
- **`python.Dockerfile.tmpl`** - Python applications

## Template Variables

Each template has access to the following variables from the project settings:

- `{{.InstallCommand}}` - Command to install dependencies (e.g., `npm ci`, `pip install -r requirements.txt`)
- `{{.BuildCommand}}` - Command to build the application (e.g., `npm run build`, `go build`)
- `{{.RunCommand}}` - Command to run the application (e.g., `npm start`, `python app.py`)
- `{{.Port}}` - Port to expose (defaults to `8080`)

## How Templates Work

1. When a deployment is created, the Builder Service selects the appropriate template based on the project's language
2. The template is populated with commands from the project settings
3. A Dockerfile is generated and written to the repository
4. Docker builds the image using this generated Dockerfile

## Editing Templates

You can edit these templates to customize the build process for each language:

### Example: Adding Environment Variables

```dockerfile
# In node.Dockerfile.tmpl, add:
ENV NODE_ENV=production
ENV API_URL=https://api.example.com
```

### Example: Installing System Dependencies

```dockerfile
# In python.Dockerfile.tmpl, add:
RUN apt-get update && apt-get install -y \
    postgresql-client \
    redis-tools \
    && rm -rf /var/lib/apt/lists/*
```

### Example: Changing Base Image Version

```dockerfile
# Change from:
FROM node:18-alpine AS builder

# To:
FROM node:20-alpine AS builder
```

## Template Embedding

These templates are embedded into the Go binary at compile time using `//go:embed`. This means:

- ✅ Templates are always available with the binary
- ✅ No external file dependencies at runtime
- ⚠️ You must **rebuild** the binary after editing templates

```bash
# After editing templates, rebuild:
go build -o bin/server ./cmd/server
```

## Template Best Practices

### Multi-stage Builds

All templates use multi-stage builds to:

- Keep final images small
- Separate build dependencies from runtime dependencies
- Cache layers efficiently

### Example Structure

```dockerfile
# Stage 1: Build
FROM <base-image> AS builder
WORKDIR /app
# Install build dependencies
# Copy source code
# Build application

# Stage 2: Production
FROM <minimal-base-image>
WORKDIR /app
# Copy only built artifacts
# Install only runtime dependencies
CMD [run command]
```

### Security

- Use specific image versions (not `latest`)
- Run as non-root user when possible (see Next.js template)
- Remove unnecessary files in production stage
- Use `.dockerignore` to exclude files

### Optimization

- Copy `package.json` before source code (cache dependencies)
- Use `--only=production` for production dependencies
- Remove build tools in production stage
- Use alpine/slim images for smaller size

## Testing Templates

To test a template change:

```bash
# 1. Edit the template file
vim templates/node.Dockerfile.tmpl

# 2. Rebuild the server
go build -o bin/server ./cmd/server

# 3. Restart the server
./bin/server

# 4. Create a test deployment
curl -X POST http://localhost:8080/api/v1/deployments \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "project_id": "...",
    "commit_hash": "...",
    "branch": "main"
  }'

# 5. Check the generated Dockerfile and logs
cat temp/*.log
```

## Common Customizations

### Node.js: Use Yarn Instead of npm

```dockerfile
# In node.Dockerfile.tmpl
COPY package.json yarn.lock ./
RUN {{.InstallCommand}}  # User sets this to "yarn install"
```

### Python: Use Poetry

```dockerfile
# In python.Dockerfile.tmpl
COPY pyproject.toml poetry.lock ./
RUN pip install poetry && \
    poetry config virtualenvs.create false && \
    {{.InstallCommand}}  # User sets to "poetry install --no-dev"
```

### Go: Multi-binary Support

```dockerfile
# In go.Dockerfile.tmpl
RUN {{.BuildCommand}}  # User sets to "CGO_ENABLED=0 go build -o bin/app ./cmd/app"
COPY --from=builder /app/bin/app /app/app
CMD ["/app/app"]
```

## Adding New Language Templates

To add support for a new language:

1. Create new template file: `templates/newlang.Dockerfile.tmpl`
2. Update `template_generator.go`:

```go
templateMap := map[project.Language]string{
    // ... existing templates ...
    project.LanguageNewLang: "templates/newlang.Dockerfile.tmpl",
}
```

3. Add the language to the domain layer constants
4. Rebuild and test

## Troubleshooting

### Template Not Found Error

```
failed to load template templates/node.Dockerfile.tmpl: file does not exist
```

**Solution**: Rebuild the binary to re-embed templates

### Invalid Template Syntax

```
failed to execute template: template: dockerfile:5: unexpected "}"
```

**Solution**: Check your Go template syntax. Use `{{.Variable}}` not `{.Variable}`

### Build Fails with Generated Dockerfile

Check the deployment logs in `temp/` directory to see the generated Dockerfile and build output.

## File Naming Convention

Template files must follow this naming pattern:

- Format: `<language>.Dockerfile.tmpl`
- Lowercase language name
- Must match the filename in `template_generator.go`
- Extension must be `.tmpl` for embedding

## Further Reading

- [Go text/template documentation](https://pkg.go.dev/text/template)
- [Docker multi-stage builds](https://docs.docker.com/build/building/multi-stage/)
- [Dockerfile best practices](https://docs.docker.com/develop/develop-images/dockerfile_best-practices/)

