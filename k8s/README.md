# Kubernetes Manifests

This directory contains Kubernetes manifests for deploying Posthoot to GKE.

## Files

- `redis-deployment.yaml` - Redis deployment and service
- `app-deployment.yaml` - Application deployment and service
- `configmap.yaml` - Non-sensitive configuration
- `secrets.yaml.example` - Example secrets file (copy to `secrets.yaml` and fill in values)

## Usage

These manifests can be used independently of Terraform, or they serve as reference for the Terraform-managed resources.

### Manual Deployment

1. **Create namespace**:
   ```bash
   kubectl create namespace posthoot
   ```

2. **Create secrets**:
   ```bash
   cp secrets.yaml.example secrets.yaml
   # Edit secrets.yaml with your values
   kubectl apply -f secrets.yaml
   ```

3. **Create ConfigMap**:
   ```bash
   kubectl apply -f configmap.yaml
   ```

4. **Deploy Redis**:
   ```bash
   kubectl apply -f redis-deployment.yaml
   ```

5. **Deploy Application**:
   ```bash
   # Update the image reference in app-deployment.yaml
   kubectl apply -f app-deployment.yaml
   ```

### Using with Terraform

The Terraform configuration in `../terraform/` automatically manages these resources. You don't need to apply these manifests manually when using Terraform.

## Notes

- The application expects a `/health` endpoint for health checks
- Redis is deployed as a small pod with minimal resources (128Mi memory request, 256Mi limit)
- The application service uses LoadBalancer type to get an external IP
- All secrets should be managed securely (use GCP Secret Manager or similar in production)

