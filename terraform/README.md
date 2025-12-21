# GKE Terraform Deployment

This directory contains Terraform configuration to deploy the Posthoot application to Google Kubernetes Engine (GKE) with Redis.

## Prerequisites

1. **Google Cloud SDK**: Install and configure `gcloud` CLI
   ```bash
   gcloud auth login
   gcloud config set project YOUR_PROJECT_ID
   ```

2. **Terraform**: Install Terraform >= 1.0
   ```bash
   brew install terraform  # macOS
   # or download from https://www.terraform.io/downloads
   ```

3. **Docker**: For building and pushing container images

4. **GCP Service Account**: Create a service account with the following roles:
   - Kubernetes Engine Admin
   - Service Account User
   - Storage Admin (for GCR)

## Setup

1. **Copy the example variables file**:
   ```bash
   cp terraform.tfvars.example terraform.tfvars
   ```

2. **Edit `terraform.tfvars`** with your configuration:
   - Set `project_id` to your GCP project ID
   - Set `github_owner` to your GitHub username or organization
   - Set `image_registry` to `"ghcr.io"` for GitHub Container Registry
   - Set `image_pull_secret_enabled = false` for public images
   - Configure database credentials (or use defaults and update via kubectl later)
   - Set JWT secret (or use default and update via kubectl later)

**For public images on ghcr.io**: No authentication needed! Just set `image_pull_secret_enabled = false`.

**For private images**: You'll need to set `image_pull_secret_enabled = true` and provide GitHub credentials.

## Building and Pushing Docker Image

### Using GitHub Container Registry (ghcr.io) - Recommended

If your image is public on GitHub Container Registry, you don't need image pull secrets:

```bash
# Set your GitHub username/org
export GITHUB_OWNER=your-github-username-or-org
export GITHUB_TOKEN=your-github-personal-access-token

# Build and push (uses latest tag by default)
./scripts/build-and-push.sh

# Or specify a tag
./scripts/build-and-push.sh v1.0.0
```

Or manually:

```bash
# Build
docker build -t ghcr.io/YOUR_GITHUB_OWNER/posthoot:latest .

# Login to GitHub Container Registry
echo "$GITHUB_TOKEN" | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin

# Push
docker push ghcr.io/YOUR_GITHUB_OWNER/posthoot:latest
```

**Note**: For public images, set `image_pull_secret_enabled = false` in your `terraform.tfvars`.

### Using Google Container Registry (gcr.io)

```bash
# Set your GCP project ID
export GCP_PROJECT_ID=your-project-id

# Build and push to GCR
./scripts/build-and-push.sh latest gcr.io

# Or manually
docker build -t gcr.io/YOUR_PROJECT_ID/posthoot:latest .
gcloud auth configure-docker gcr.io --quiet
docker push gcr.io/YOUR_PROJECT_ID/posthoot:latest
```

## Deployment

1. **Initialize Terraform**:
   ```bash
   cd terraform
   terraform init
   ```

2. **Plan the deployment**:
   ```bash
   terraform plan
   ```

3. **Apply the configuration**:
   ```bash
   terraform apply
   ```

   This will create:
   - GKE cluster
   - Node pool
   - Namespace
   - Redis deployment and service
   - Application deployment and service
   - ConfigMaps and Secrets

4. **Get cluster credentials**:
   ```bash
   gcloud container clusters get-credentials posthoot-cluster \
     --region us-central1 \
     --project YOUR_PROJECT_ID
   ```

5. **Verify deployment**:
   ```bash
   kubectl get pods -n posthoot
   kubectl get services -n posthoot
   ```

6. **Get LoadBalancer IP**:
   ```bash
   kubectl get service posthoot -n posthoot
   ```

   The `EXTERNAL-IP` will be available after a few minutes.

## Configuration

### Environment Variables

The application uses environment variables configured through:
- **ConfigMap** (`posthoot-config`): Non-sensitive configuration
- **Secrets** (`posthoot-secrets`): Sensitive data (passwords, keys)

### Updating Configuration

1. **Update ConfigMap**:
   ```bash
   kubectl edit configmap posthoot-config -n posthoot
   ```

2. **Update Secrets**:
   ```bash
   kubectl edit secret posthoot-secrets -n posthoot
   ```

3. **Restart pods** to apply changes:
   ```bash
   kubectl rollout restart deployment/posthoot -n posthoot
   ```

### Scaling

Scale the application:
```bash
kubectl scale deployment posthoot --replicas=3 -n posthoot
```

Or update `app_replicas` in `terraform.tfvars` and run `terraform apply`.

## Monitoring

Check logs:
```bash
# Application logs
kubectl logs -f deployment/posthoot -n posthoot

# Redis logs
kubectl logs -f deployment/redis -n posthoot
```

Check pod status:
```bash
kubectl get pods -n posthoot
kubectl describe pod <pod-name> -n posthoot
```

## Troubleshooting

### Pods not starting
```bash
# Check pod events
kubectl describe pod <pod-name> -n posthoot

# Check logs
kubectl logs <pod-name> -n posthoot
```

### Image pull errors
- For ghcr.io: Verify image is public or image pull secret is configured correctly
- Check image exists: Visit `https://github.com/YOUR_OWNER?tab=packages` or use `docker pull ghcr.io/YOUR_OWNER/posthoot:latest`
- Verify image pull secret (if using private images): `kubectl get secret registry-pull-secret -n posthoot`
- For gcr.io: Verify service account key is correct and check image exists: `gcloud container images list --repository=gcr.io/YOUR_PROJECT_ID`

### Connection issues
- Verify Redis service: `kubectl get svc redis -n posthoot`
- Check DNS resolution: `kubectl exec -it <pod-name> -n posthoot -- nslookup redis`

## Cleanup

To destroy all resources:
```bash
terraform destroy
```

**Warning**: This will delete the entire GKE cluster and all resources!

## Cost Optimization

- Use preemptible nodes: Set `use_preemptible_nodes = true`
- Adjust node pool size: Reduce `min_node_count` and `max_node_count`
- Use smaller machine types: Change `machine_type` to `e2-small` or `e2-micro`
- Enable cluster autoscaling: Already configured with min/max node counts

## Security Best Practices

1. **Enable private cluster**: Set `enable_private_cluster = true`
2. **Use Workload Identity**: Already configured
3. **Rotate secrets regularly**: Update secrets in Kubernetes
4. **Enable network policies**: Already enabled
5. **Use least privilege**: Review service account permissions
6. **Enable audit logging**: Configure in GCP Console

## Additional Resources

- [GKE Documentation](https://cloud.google.com/kubernetes-engine/docs)
- [Terraform GKE Provider](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/container_cluster)
- [Kubernetes Documentation](https://kubernetes.io/docs/)

