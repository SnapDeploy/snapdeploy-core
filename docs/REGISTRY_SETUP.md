# Docker Registry Setup Guide

This guide helps you choose and configure a Docker registry for SnapDeploy.

## Quick Comparison

| Registry                  | Best For               | Free Tier      | Cost           | Setup Time |
| ------------------------- | ---------------------- | -------------- | -------------- | ---------- |
| GitHub Container Registry | Startups, GitHub users | 500MB private  | FREE           | 5 min      |
| AWS ECR                   | Production, AWS users  | None           | ~$1-5/mo       | 10 min     |
| Google Artifact Registry  | GCP users              | None           | ~$1-5/mo       | 10 min     |
| Docker Hub                | Quick testing          | 1 private repo | $0-9/mo        | 5 min      |
| Harbor (Self-hosted)      | Full control           | Unlimited      | Infrastructure | 30 min     |

---

## 1. GitHub Container Registry (RECOMMENDED FOR STARTING)

### Why Choose This?

- ✅ FREE 500MB storage for private images
- ✅ Unlimited public images
- ✅ No rate limits for authenticated users
- ✅ Perfect for startups and testing
- ✅ Easiest to set up

### Setup Steps

1. **Create GitHub Personal Access Token**

   - Go to: https://github.com/settings/tokens
   - Click "Generate new token (classic)"
   - Select scopes: `read:packages`, `write:packages`, `delete:packages`
   - Copy the token

2. **Configure Environment**

   ```bash
   # Add to your .env or environment
   DOCKER_REGISTRY=ghcr.io/your-github-username
   DOCKER_REGISTRY_USERNAME=your-github-username
   DOCKER_REGISTRY_PASSWORD=ghp_xxxxxxxxxxxxxxxxxxxxx  # Your token
   ```

3. **Login to Registry**

   ```bash
   echo $DOCKER_REGISTRY_PASSWORD | docker login ghcr.io \
     -u $DOCKER_REGISTRY_USERNAME --password-stdin
   ```

4. **Update Builder Service**

   The builder will automatically use these credentials. No code changes needed!

5. **Test It**

   ```bash
   # Build a test image
   docker build -t ghcr.io/your-username/test-app:latest .

   # Push it
   docker push ghcr.io/your-username/test-app:latest
   ```

### Cost

- **Free tier**: 500MB private storage, unlimited public
- **Paid**: $0.25/GB/month beyond free tier

---

## 2. AWS ECR (RECOMMENDED FOR PRODUCTION)

### Why Choose This?

- ✅ Best for production deployments
- ✅ Seamless AWS integration
- ✅ Excellent security and compliance
- ✅ No pull rate limits
- ✅ Built-in vulnerability scanning

### Setup Steps

1. **Install AWS CLI**

   ```bash
   # On Ubuntu/Debian
   curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
   unzip awscliv2.zip
   sudo ./aws/install

   # Configure
   aws configure
   ```

2. **Create ECR Repository**

   ```bash
   # Create repository
   aws ecr create-repository \
     --repository-name snapdeploy-apps \
     --region us-east-1

   # Enable image scanning
   aws ecr put-image-scanning-configuration \
     --repository-name snapdeploy-apps \
     --image-scanning-configuration scanOnPush=true \
     --region us-east-1
   ```

3. **Get Registry URL**

   ```bash
   aws ecr describe-repositories \
     --repository-names snapdeploy-apps \
     --region us-east-1 \
     --query 'repositories[0].repositoryUri' \
     --output text

   # Example output: 123456789.dkr.ecr.us-east-1.amazonaws.com/snapdeploy-apps
   ```

4. **Configure Environment**

   ```bash
   # Add to .env
   DOCKER_REGISTRY=123456789.dkr.ecr.us-east-1.amazonaws.com
   AWS_REGION=us-east-1
   AWS_ACCOUNT_ID=123456789
   ```

5. **Update Builder to Use ECR Authentication**

   Add this helper function to your builder:

   ```go
   // In docker_client.go
   func (dc *DockerClient) LoginECR(ctx context.Context, region string) error {
       cmd := exec.Command("aws", "ecr", "get-login-password", "--region", region)
       password, err := cmd.Output()
       if err != nil {
           return fmt.Errorf("failed to get ECR password: %w", err)
       }

       registry := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com",
           os.Getenv("AWS_ACCOUNT_ID"), region)

       authConfig := registry.AuthConfig{
           Username: "AWS",
           Password: string(password),
       }

       _, err = dc.client.RegistryLogin(ctx, authConfig)
       return err
   }
   ```

6. **Login to ECR**
   ```bash
   aws ecr get-login-password --region us-east-1 | \
     docker login --username AWS --password-stdin \
     123456789.dkr.ecr.us-east-1.amazonaws.com
   ```

### Cost

- **Storage**: $0.10/GB/month
- **Data Transfer**: Free within AWS, $0.09/GB outbound
- **Example**: 10GB of images = ~$1/month

### IAM Policy for Builder

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ecr:GetAuthorizationToken",
        "ecr:BatchCheckLayerAvailability",
        "ecr:GetDownloadUrlForLayer",
        "ecr:BatchGetImage",
        "ecr:PutImage",
        "ecr:InitiateLayerUpload",
        "ecr:UploadLayerPart",
        "ecr:CompleteLayerUpload"
      ],
      "Resource": "*"
    }
  ]
}
```

---

## 3. Harbor (Self-Hosted)

### Why Choose This?

- ✅ Complete control over your infrastructure
- ✅ Unlimited storage (your hardware)
- ✅ FREE and open-source
- ✅ Enterprise features (RBAC, replication, scanning)

### Setup with Docker Compose

1. **Download Harbor**

   ```bash
   wget https://github.com/goharbor/harbor/releases/download/v2.10.0/harbor-online-installer-v2.10.0.tgz
   tar xvf harbor-online-installer-v2.10.0.tgz
   cd harbor
   ```

2. **Configure Harbor**

   ```bash
   cp harbor.yml.tmpl harbor.yml

   # Edit harbor.yml
   vim harbor.yml
   ```

   Update these settings:

   ```yaml
   hostname: registry.yourdomain.com
   http:
     port: 80
   https:
     port: 443
     certificate: /path/to/cert.crt
     private_key: /path/to/cert.key
   harbor_admin_password: YourSecurePassword123
   database:
     password: DatabasePassword123
   data_volume: /data/harbor
   ```

3. **Install Harbor**

   ```bash
   sudo ./install.sh --with-trivy --with-chartmuseum
   ```

4. **Configure Environment**

   ```bash
   # Add to .env
   DOCKER_REGISTRY=registry.yourdomain.com
   DOCKER_REGISTRY_USERNAME=admin
   DOCKER_REGISTRY_PASSWORD=YourSecurePassword123
   ```

5. **Create Project in Harbor**

   - Access: https://registry.yourdomain.com
   - Login with admin credentials
   - Create new project: "snapdeploy-apps"

6. **Login**
   ```bash
   docker login registry.yourdomain.com
   ```

### Cost

- **Software**: FREE
- **Infrastructure**:
  - Small setup: $5-20/month VPS
  - Production: $50-200/month (dedicated server)
  - Storage: Based on your disk space

---

## 4. Docker Hub

### Why Choose This?

- ✅ Familiar to most developers
- ✅ Quick setup
- ✅ Good for public images

### Setup Steps

1. **Create Account**

   - Go to: https://hub.docker.com/signup
   - Create account

2. **Create Access Token**

   - Account Settings → Security → New Access Token
   - Copy the token

3. **Configure Environment**

   ```bash
   # Add to .env
   DOCKER_REGISTRY=docker.io/your-username
   # or just:
   DOCKER_REGISTRY=your-username  # Docker Hub is default
   DOCKER_REGISTRY_USERNAME=your-username
   DOCKER_REGISTRY_PASSWORD=dckr_pat_xxxxxxxxxxxxx
   ```

4. **Login**
   ```bash
   docker login -u your-username -p dckr_pat_xxxxxxxxxxxxx
   ```

### Cost

- **Free**: 1 private repo, 100 pulls/6hrs
- **Pro** ($5/mo): 5 private repos, unlimited pulls
- **Team** ($9/mo): Unlimited private repos, team features

### Limitations

- ⚠️ Rate limits on free tier
- ⚠️ Only 1 free private repo

---

## 5. Google Artifact Registry

### Setup Steps

1. **Install gcloud CLI**

   ```bash
   curl https://sdk.cloud.google.com | bash
   gcloud init
   ```

2. **Enable Artifact Registry API**

   ```bash
   gcloud services enable artifactregistry.googleapis.com
   ```

3. **Create Repository**

   ```bash
   gcloud artifacts repositories create snapdeploy-apps \
     --repository-format=docker \
     --location=us-central1 \
     --description="SnapDeploy application images"
   ```

4. **Configure Docker**

   ```bash
   gcloud auth configure-docker us-central1-docker.pkg.dev
   ```

5. **Configure Environment**
   ```bash
   DOCKER_REGISTRY=us-central1-docker.pkg.dev/project-id/snapdeploy-apps
   ```

### Cost

- **Storage**: $0.10/GB/month
- **Network**: $0.12/GB egress

---

## Integration with Builder Service

### Environment Configuration

Add to your `.env` file:

```bash
# Registry Configuration
DOCKER_REGISTRY=ghcr.io/your-username  # or your chosen registry
DOCKER_REGISTRY_USERNAME=your-username
DOCKER_REGISTRY_PASSWORD=your-token-or-password

# For AWS ECR (optional)
AWS_REGION=us-east-1
AWS_ACCOUNT_ID=123456789

# Builder Configuration
BUILDER_WORK_DIR=/tmp/snapdeploy/builds
BUILDER_LOG_DIR=./temp
```

### Update Builder to Use Registry Credentials

Add authentication to `docker_client.go`:

```go
package builder

import (
    "context"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "os"

    "github.com/docker/docker/api/types/registry"
)

// GetRegistryAuth returns authentication config for the registry
func GetRegistryAuth() (string, error) {
    username := os.Getenv("DOCKER_REGISTRY_USERNAME")
    password := os.Getenv("DOCKER_REGISTRY_PASSWORD")

    if username == "" || password == "" {
        return "", nil // No auth configured
    }

    authConfig := registry.AuthConfig{
        Username: username,
        Password: password,
    }

    encodedJSON, err := json.Marshal(authConfig)
    if err != nil {
        return "", err
    }

    return base64.URLEncoding.EncodeToString(encodedJSON), nil
}

// Update PushImage to use authentication
func (dc *DockerClient) PushImage(ctx context.Context, imageName string) (io.ReadCloser, error) {
    registryAuth, err := GetRegistryAuth()
    if err != nil {
        return nil, fmt.Errorf("failed to get registry auth: %w", err)
    }

    pushOpts := image.PushOptions{
        RegistryAuth: registryAuth,
    }

    response, err := dc.client.ImagePush(ctx, imageName, pushOpts)
    if err != nil {
        return nil, fmt.Errorf("failed to push image: %w", err)
    }

    return response, nil
}
```

---

## Testing Your Registry Setup

```bash
# 1. Test Docker login
docker login $DOCKER_REGISTRY

# 2. Build a test image
echo "FROM alpine:latest" > Dockerfile.test
echo "CMD echo 'Hello from SnapDeploy'" >> Dockerfile.test
docker build -t $DOCKER_REGISTRY/test:latest -f Dockerfile.test .

# 3. Push the test image
docker push $DOCKER_REGISTRY/test:latest

# 4. Pull it back
docker pull $DOCKER_REGISTRY/test:latest

# 5. Run it
docker run $DOCKER_REGISTRY/test:latest

# Success! Your registry is working
```

---

## Recommendations by Use Case

### Hobbyist / MVP

→ **GitHub Container Registry** (FREE, easy)

### Startup / Small Team

→ **GitHub Container Registry** or **AWS ECR** ($1-5/month)

### Growing Company

→ **AWS ECR** or **Google Artifact Registry** ($10-50/month)

### Enterprise / Compliance

→ **Harbor (Self-hosted)** or **AWS ECR Private** (full control)

### Multi-cloud

→ **Harbor (Self-hosted)** with replication

---

## Security Best Practices

1. **Use Access Tokens**, not passwords
2. **Enable Image Scanning** for vulnerabilities
3. **Use Private Registries** for production
4. **Rotate Credentials** regularly
5. **Implement RBAC** (Role-Based Access Control)
6. **Enable Audit Logging**
7. **Use HTTPS** always
8. **Tag Images Properly** (don't use `:latest` in production)

---

## Monitoring

### Track These Metrics

- **Storage usage** - cost management
- **Push/pull rates** - performance
- **Failed pushes** - deployment issues
- **Vulnerability scan results** - security

### Example: AWS ECR Monitoring

```bash
# Get repository metrics
aws ecr describe-repositories --repository-names snapdeploy-apps

# List images
aws ecr list-images --repository-name snapdeploy-apps

# Get image scan results
aws ecr describe-image-scan-findings \
  --repository-name snapdeploy-apps \
  --image-id imageTag=latest
```

---

## Troubleshooting

### "unauthorized: authentication required"

→ Run `docker login` with correct credentials

### "denied: requested access to the resource is denied"

→ Check registry permissions and repository name

### "manifest blob unknown"

→ Image doesn't exist, check image name and tag

### "connection refused"

→ Check registry URL and network connectivity

### "rate limit exceeded" (Docker Hub)

→ Authenticate or upgrade plan

---

## Cost Estimation Tool

Calculate your expected costs:

```
Storage needed: ____ GB × $0.10/GB = $____/month
Images per project: ____
Number of projects: ____
Total storage: ____ GB

GitHub Container Registry: FREE (up to 500MB)
AWS ECR: ~$____ /month
Docker Hub Pro: $5/month (unlimited private repos)
Harbor: $____ /month (infrastructure only)
```

---

## Next Steps

1. Choose your registry based on the comparison above
2. Follow the setup steps for your chosen registry
3. Update your `.env` file with registry credentials
4. Update `docker_client.go` to use authentication
5. Test with a deployment
6. Monitor usage and costs

For production, I recommend starting with **GitHub Container Registry** (free) and migrating to **AWS ECR** as you scale.

