# Posthoot Server Kubernetes Setup

This directory contains Kubernetes manifests for deploying the Posthoot server application.

## Prerequisites

- Kubernetes cluster (v1.20+)
- kubectl configured to access your cluster
- Docker registry access (for pushing the server image)
- NGINX Ingress Controller installed
- cert-manager installed (for SSL certificates)

## Architecture

The setup includes:

- **Namespace**: `posthoot-server` for isolation
- **Database**: PostgreSQL with persistent storage
- **Cache**: Redis with persistent storage
- **Application**: Posthoot server (2 replicas)
- **Monitoring**: Asynqmon for background job monitoring
- **Networking**: Ingress with SSL termination

## Quick Start

### 1. Build and Push Docker Image

```bash
# Build the image
docker build -t posthoot-server:latest .

# Tag for your registry
docker tag posthoot-server:latest your-registry/posthoot-server:latest

# Push to registry
docker push your-registry/posthoot-server:latest
```

### 2. Update Secrets

Edit `secret.yaml` and replace the base64-encoded values with your actual secrets:

```bash
# Generate base64 values
echo -n "your-actual-password" | base64
```

### 3. Update Configuration

- Edit `configmap.yaml` for non-sensitive configuration
- Update `ingress.yaml` with your domain name
- Modify resource limits in deployments as needed

### 4. Deploy

```bash
# Apply all resources
kubectl apply -k .

# Or apply individually
kubectl apply -f namespace.yaml
kubectl apply -f configmap.yaml
kubectl apply -f secret.yaml
# ... etc
```

### 5. Verify Deployment

```bash
# Check all resources
kubectl get all -n posthoot-server

# Check pods
kubectl get pods -n posthoot-server

# Check services
kubectl get svc -n posthoot-server

# Check ingress
kubectl get ingress -n posthoot-server
```

## Configuration

### Environment Variables

The application uses the following environment variables:

**Server Configuration:**
- `SERVER_HOST`: Server host (default: 0.0.0.0)
- `SERVER_PORT`: Server port (default: 9001)

**Database Configuration:**
- `DB_HOST`: PostgreSQL host
- `DB_PORT`: PostgreSQL port
- `DB_NAME`: Database name
- `DB_USER`: Database user
- `POSTGRES_PASSWORD`: Database password

**Redis Configuration:**
- `REDIS_HOST`: Redis host
- `REDIS_PORT`: Redis port
- `REDIS_PASSWORD`: Redis password (optional)

**Application Settings:**
- `ENVIRONMENT`: Environment (production/staging/development)
- `LOG_LEVEL`: Log level (debug/info/warn/error)
- `JWT_SECRET`: JWT signing secret

### Resource Limits

Current resource allocations:

- **PostgreSQL**: 256Mi-512Mi RAM, 250m-500m CPU
- **Redis**: 128Mi-256Mi RAM, 100m-200m CPU
- **Server**: 512Mi-1Gi RAM, 500m-1000m CPU
- **Asynqmon**: 128Mi-256Mi RAM, 100m-200m CPU

### Storage

- **PostgreSQL**: 10Gi persistent storage
- **Redis**: 5Gi persistent storage

## Monitoring

### Health Checks

All deployments include health checks:

- **Server**: HTTP GET `/health` endpoint
- **PostgreSQL**: `pg_isready` command
- **Redis**: `redis-cli ping` command

### Background Job Monitoring

Asynqmon is deployed to monitor background jobs:
- Access via: `http://posthoot-asynqmon:8080`
- Requires Redis connection to `posthoot-redis:6379`

## Scaling

### Horizontal Pod Autoscaling

To enable HPA, create an HPA resource:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: posthoot-server-hpa
  namespace: posthoot-server
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: posthoot-server
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

### Vertical Pod Autoscaling

For VPA, install the VPA controller and create VPA resources.

## Troubleshooting

### Common Issues

1. **Image Pull Errors**
   - Ensure image is pushed to registry
   - Check imagePullPolicy and image name

2. **Database Connection Issues**
   - Verify PostgreSQL is running: `kubectl logs -n posthoot-server deployment/posthoot-postgres`
   - Check secrets are properly configured

3. **Redis Connection Issues**
   - Verify Redis is running: `kubectl logs -n posthoot-server deployment/posthoot-redis`
   - Check network policies

4. **Ingress Issues**
   - Verify NGINX Ingress Controller is installed
   - Check domain DNS resolution
   - Verify SSL certificate generation

### Debugging Commands

```bash
# Check pod logs
kubectl logs -n posthoot-server deployment/posthoot-server

# Check pod status
kubectl describe pod -n posthoot-server <pod-name>

# Check events
kubectl get events -n posthoot-server --sort-by='.lastTimestamp'

# Port forward for local access
kubectl port-forward -n posthoot-server svc/posthoot-server 9001:80
```

## Backup and Recovery

### Database Backup

```bash
# Create backup
kubectl exec -n posthoot-server deployment/posthoot-postgres -- pg_dump -U posthoot posthoot > backup.sql

# Restore backup
kubectl exec -n posthoot-server deployment/posthoot-postgres -- psql -U posthoot posthoot < backup.sql
```

### Persistent Volume Backup

Consider using Velero or similar tools for PV backup.

## Security

### Network Policies

Consider implementing network policies to restrict pod-to-pod communication.

### RBAC

The setup uses default RBAC. Consider creating specific ServiceAccounts and Roles for better security.

### Secrets Management

For production, consider using:
- External secrets operator
- HashiCorp Vault
- AWS Secrets Manager
- Azure Key Vault

## Updates and Maintenance

### Rolling Updates

```bash
# Update image
kubectl set image deployment/posthoot-server posthoot-server=your-registry/posthoot-server:v1.1.0 -n posthoot-server

# Monitor rollout
kubectl rollout status deployment/posthoot-server -n posthoot-server
```

### Database Migrations

Handle database migrations through the application or separate migration jobs.

## Support

For issues and questions:
- Check application logs
- Review Kubernetes events
- Verify configuration values
- Test connectivity between services
