# Deployment Strategy - ECS with Custom Subdomains

## Overview

SnapDeploy uses an **Update-in-Place** deployment strategy where each project gets:
- One active ECS Fargate service per project
- A custom subdomain (e.g., `my-app.snapdeploy.app`)
- Automatic DNS configuration via Route53
- Zero-downtime rolling updates

## Architecture

```
┌─────────────┐
│   User      │
│  Creates    │──┐
│ Deployment  │  │
└─────────────┘  │
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│                    Build Phase                          │
│  ┌─────────┐    ┌──────────┐    ┌─────────────┐       │
│  │ Generate│───▶│CodeBuild │───▶│Push to ECR  │       │
│  │Dockerfile│   │Builds    │    │Registry     │       │
│  └─────────┘    │& Tests   │    └─────────────┘       │
│                 └──────────┘                            │
└─────────────────────────────────────────────────────────┘
                 │
                 │ On Success
                 ▼
┌─────────────────────────────────────────────────────────┐
│                 Deployment Phase                        │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐         │
│  │  Create  │───▶│ Update   │───▶│Configure │         │
│  │   ECS    │    │  Task    │    │ Route53  │         │
│  │ Service  │    │   Def    │    │   DNS    │         │
│  └──────────┘    └──────────┘    └──────────┘         │
└─────────────────────────────────────────────────────────┘
                 │
                 │ Deployed
                 ▼
         https://my-app.snapdeploy.app
```

## Cost Breakdown (per deployment)

| Component | Specs | Cost/Hour | Cost/Month |
|-----------|-------|-----------|------------|
| ECS Fargate | 0.25 vCPU, 512MB RAM | ~$0.02 | ~$15 |
| ALB (shared) | Amortized over all projects | ~$0.005 | ~$4 |
| Route53 Records | Per domain | $0.00002 | $0.50 |
| **Total per deployment** | | **~$0.025** | **~$19.50** |

**Maximum budget**: 100 deployments × $0.025/hour = **$2.50/hour** (well under $1/hour per deployment ✅)

## Custom Domain Configuration

### Domain Format
```
<custom-prefix>.snapdeploy.app
```

Examples:
- `my-app.snapdeploy.app`
- `staging-v2.snapdeploy.app`
- `abc12345.snapdeploy.app` (auto-generated if not specified)

### Domain Validation Rules
- Length: 1-63 characters
- Characters: lowercase letters, numbers, hyphens
- Must start and end with alphanumeric character
- Reserved: `www`, `api`, `admin`, `app`, `dashboard`, etc.

### SSL/TLS Certificates

Use **AWS Certificate Manager (ACM) wildcard certificate**:

```bash
# Request wildcard certificate
aws acm request-certificate \
  --domain-name "*.snapdeploy.app" \
  --domain-name "snapdeploy.app" \
  --validation-method DNS \
  --region us-east-1
```

**Benefits**:
- ✅ FREE with AWS
- ✅ Covers unlimited subdomains
- ✅ Auto-renewal
- ✅ Integrated with ALB

## Infrastructure Setup

### 1. Route53 Hosted Zone

```bash
# Create hosted zone for your domain
aws route53 create-hosted-zone \
  --name snapdeploy.app \
  --caller-reference $(date +%s)

# Note the hosted zone ID for configuration
```

### 2. Application Load Balancer (ALB)

**Configuration**:
- Listener: HTTPS (443) with wildcard certificate
- Default action: Fixed response (404)
- Host-based routing rules dynamically created per project

**Routing Rule Example**:
```
IF host = "my-app.snapdeploy.app" 
THEN forward to target-group-my-app
```

### 3. ECS Cluster

```bash
# Create ECS cluster
aws ecs create-cluster \
  --cluster-name snapdeploy-cluster \
  --capacity-providers FARGATE \
  --default-capacity-provider-strategy \
    capacityProvider=FARGATE,weight=1
```

### 4. VPC & Security Groups

**Requirements**:
- Public subnets in 2+ AZs (for high availability)
- Security group allowing:
  - Inbound: Port 8080 from ALB security group
  - Outbound: All traffic (for ECR, internet, etc.)

## Deployment Flow

### 1. User Creates Project

```json
POST /api/v1/users/:id/projects
{
  "repository_url": "https://github.com/user/repo",
  "language": "NODE",
  "custom_domain": "my-app",  // Optional - auto-generated if empty
  "install_command": "npm install",
  "build_command": "npm run build",
  "run_command": "npm start"
}
```

**Response includes**:
```json
{
  "custom_domain": "my-app",
  "deployment_url": "https://my-app.snapdeploy.app"
}
```

### 2. User Triggers Deployment

```json
POST /api/v1/deployments
{
  "project_id": "uuid",
  "branch": "main",
  "commit_hash": "abc123"
}
```

### 3. Build Phase (CodeBuild)

1. Clone repository
2. Generate Dockerfile from template
3. Build Docker image
4. Push to ECR: `123456789.dkr.ecr.us-east-1.amazonaws.com/snapdeploy-abc12345:commit-hash`
5. Trigger deployment callback

### 4. Deploy Phase (ECS + Route53)

1. **Create/Update Task Definition**
   - Image: From CodeBuild
   - CPU: 256 (0.25 vCPU)
   - Memory: 512 MB
   - Port: 8080
   - Environment variables

2. **Create/Update ECS Service**
   - Service name: `snapdeploy-{project-id-prefix}`
   - Desired count: 1
   - Launch type: Fargate
   - Load balancer: Attach to target group
   - Rolling update strategy

3. **Configure DNS**
   - Create/Update Route53 A record (ALIAS to ALB)
   - Domain: `{custom-domain}.snapdeploy.app`
   - Target: ALB DNS name
   - Propagation: ~60 seconds

4. **Wait for Stability**
   - Health checks: 60 second grace period
   - ECS service stabilization: ~2-5 minutes
   - Mark deployment as `DEPLOYED`

## Deployment Status Flow

```
PENDING → BUILDING → DEPLOYING → DEPLOYED
   ↓         ↓           ↓
FAILED ←──────┴───────────┘
```

## Rollback Strategy

Since we use update-in-place, rollback is achieved by:

1. **Redeploy previous commit**:
   ```bash
   POST /api/v1/deployments
   {
     "project_id": "uuid",
     "commit_hash": "previous-working-commit"
   }
   ```

2. **ECS handles rolling update**:
   - New tasks start with old image
   - Health checks pass
   - Old tasks drain and stop
   - Total time: ~3-5 minutes

## Monitoring & Logs

### CloudWatch Logs

All container logs automatically sent to:
```
/ecs/snapdeploy-{project-id}
```

### ECS Events

Monitor service events:
```bash
aws ecs describe-services \
  --cluster snapdeploy-cluster \
  --services snapdeploy-abc12345 \
  --query 'services[0].events[:10]'
```

### Deployment Logs

Real-time streaming via SSE:
```
GET /api/v1/deployments/:id/logs/stream
```

## Scaling Considerations

### Horizontal Scaling

For high-traffic projects, increase task count:

```go
DesiredCount: 3  // Instead of 1
```

### Resource Scaling

Adjust per-project based on load:

```go
CPU:    "512"   // 0.5 vCPU
Memory: "1024"  // 1 GB
```

### Auto-scaling (Future Enhancement)

```hcl
resource "aws_appautoscaling_target" "ecs_target" {
  max_capacity       = 4
  min_capacity       = 1
  resource_id        = "service/${var.cluster_name}/${var.service_name}"
  scalable_dimension = "ecs:service:DesiredCount"
  service_namespace  = "ecs"
}
```

## Cleanup & Lifecycle

### Stop Deployment
```bash
# Scale to 0 tasks
PUT /api/v1/projects/:id/deployments/stop
```

### Delete Deployment
```bash
# Remove ECS service + DNS record
DELETE /api/v1/deployments/:id
```

### Automatic Cleanup (Future)
- Stop inactive deployments after 24 hours
- Delete stopped deployments after 7 days
- Prune old ECR images (keep last 10)

## Environment Variables Required

```bash
# ECS Configuration
ECS_CLUSTER_NAME=snapdeploy-cluster
TARGET_GROUP_ARN=arn:aws:elasticloadbalancing:...
ALB_DNS_NAME=snapdeploy-alb-xxx.us-east-1.elb.amazonaws.com
SUBNET_IDS=subnet-xxx,subnet-yyy,subnet-zzz
SECURITY_GROUP_ID=sg-xxx

# Route53
ROUTE53_HOSTED_ZONE_ID=Z1234567890ABC
BASE_DOMAIN=snapdeploy.app

# AWS
AWS_REGION=us-east-1
AWS_ACCOUNT_ID=123456789012
```

## Troubleshooting

### Deployment Stuck in DEPLOYING

**Check**:
1. ECS service events (failed health checks?)
2. Task logs in CloudWatch
3. Security group rules (ALB → Container port 8080)
4. Target group health checks

**Fix**:
```bash
# Force new deployment
aws ecs update-service \
  --cluster snapdeploy-cluster \
  --service snapdeploy-abc12345 \
  --force-new-deployment
```

### DNS Not Resolving

**Check**:
1. Route53 record created correctly
2. TTL propagation (wait 60 seconds)
3. ALB DNS name is correct

**Verify**:
```bash
dig my-app.snapdeploy.app
nslookup my-app.snapdeploy.app
```

### 503 Service Unavailable

**Causes**:
- No healthy tasks running
- Target group empty
- Health check failing

**Check task health**:
```bash
aws ecs describe-tasks \
  --cluster snapdeploy-cluster \
  --tasks $(aws ecs list-tasks --cluster snapdeploy-cluster --service-name snapdeploy-abc12345 --query 'taskArns[0]' --output text)
```

## Security Considerations

### IAM Roles

**ECS Task Execution Role** (pulls images, writes logs):
```json
{
  "Effect": "Allow",
  "Action": [
    "ecr:GetAuthorizationToken",
    "ecr:BatchCheckLayerAvailability",
    "ecr:GetDownloadUrlForLayer",
    "ecr:BatchGetImage",
    "logs:CreateLogStream",
    "logs:PutLogEvents"
  ],
  "Resource": "*"
}
```

**ECS Task Role** (application permissions):
```json
{
  "Effect": "Allow",
  "Action": [
    "s3:GetObject",
    "s3:PutObject"
  ],
  "Resource": "arn:aws:s3:::user-uploads/*"
}
```

### Network Security

- ALB in public subnets (internet-facing)
- ECS tasks in private subnets (optional)
- NAT Gateway for outbound internet (ECR pulls)

## Future Enhancements

### 1. Blue-Green Deployments
- Maintain two environments
- Instant rollback by switching traffic

### 2. Preview Deployments
- Temporary deployments per PR
- Subdomain: `pr-123-my-app.snapdeploy.app`
- Auto-cleanup after PR merge

### 3. Multi-Region
- Deploy to multiple AWS regions
- Route53 geo-routing
- Lower latency worldwide

### 4. Custom Domains
- Allow users to bring their own domain
- Automatic CNAME validation
- SSL certificate provisioning

## References

- [AWS ECS Best Practices](https://docs.aws.amazon.com/AmazonECS/latest/bestpracticesguide/)
- [Route53 Routing Policies](https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/routing-policy.html)
- [ALB Target Groups](https://docs.aws.amazon.com/elasticloadbalancing/latest/application/load-balancer-target-groups.html)

