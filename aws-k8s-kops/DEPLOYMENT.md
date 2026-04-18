# Posthoot Kubernetes Deployment Guide

Production-ready Kubernetes deployment on AWS using Kops with spot instances for cost optimization.

## Architecture

- **Cluster**: Self-managed Kubernetes via Kops (avoids $73/mo EKS control plane cost)
- **Compute**: EC2 Spot Instances (60-80% cost savings)
- **Database**: Aurora PostgreSQL Serverless v2
- **Cache**: ElastiCache Serverless (Valkey/Redis compatible)
- **Storage**: S3 for state and backups
- **Monitoring**: Prometheus + Grafana + Loki
- **Ingress**: Nginx Ingress Controller with AWS NLB

## Prerequisites

### Required Tools
```bash
# Install kops
brew install kops

# Install kubectl
brew install kubectl

# Install AWS CLI
brew install awscli

# Install helm
brew install helm
```

### AWS Account Setup
1. AWS account with admin access
2. AWS CLI configured with credentials
3. S3 bucket for kops state store
4. Route53 hosted zone (optional, for DNS)

## Initial Setup

### 1. Configure AWS Credentials
```bash
aws configure --profile your-profile
# Or use: aws login --profile your-profile
```

### 2. Create Kops State Store
```bash
export AWS_PROFILE=your-profile
export AWS_REGION=us-east-2
export KOPS_STATE_STORE=s3://your-kops-state-bucket

# Create S3 bucket
aws s3 mb $KOPS_STATE_STORE --region $AWS_REGION
aws s3api put-bucket-versioning \
  --bucket $(echo $KOPS_STATE_STORE | sed 's|s3://||') \
  --versioning-configuration Status=Enabled
```

### 3. Generate SSH Key (if needed)
```bash
ssh-keygen -t ed25519 -f ~/.ssh/id_ed25519 -C "your-email@example.com"
```

## Cluster Deployment

### 1. Update Configuration

Edit `setup-x86-only.sh`:
```bash
CLUSTER_NAME=your-cluster.k8s.local
KOPS_STATE_STORE=s3://your-kops-state-bucket
AWS_REGION=us-east-2
```

### 2. Create Cluster
```bash
cd aws-k8s-kops
chmod +x setup-x86-only.sh
./setup-x86-only.sh
```

This creates:
- 1 master node (t3.small, on-demand)
- 3 worker nodes (spot instances across 3 AZs)
- Auto-scaling group with mixed instance policy

### 3. Wait for Cluster
```bash
kops validate cluster --wait 10m
```

### 4. Export Kubeconfig
```bash
kops export kubeconfig $CLUSTER_NAME --admin
```

## Install Core Components

### 1. AWS Node Termination Handler
Handles graceful spot instance termination:
```bash
kubectl apply -f install-spot-handler.yaml
```

### 2. Nginx Ingress Controller
```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/aws/deploy.yaml
```

### 3. Prometheus + Grafana Stack
```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

helm install kube-prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring --create-namespace \
  --set prometheus.prometheusSpec.retention=7d \
  --set grafana.adminPassword=admin123
```

### 4. Loki for Log Aggregation
```bash
helm repo add grafana https://grafana.github.io/helm-charts
helm install loki grafana/loki-stack \
  --namespace monitoring \
  --set grafana.enabled=false \
  --set prometheus.enabled=false \
  --set promtail.enabled=true
```

Apply Loki datasource and dashboards:
```bash
kubectl apply -f manifests/logs-dashboard.yaml
```

## Deploy Application

### 1. Create ElastiCache Serverless (Valkey)

**Option A: AWS Console**
- Service: ElastiCache Serverless
- Engine: Valkey 7.2+
- Name: your-app-valkey-serverless

**Option B: AWS CLI**
```bash
aws elasticache create-serverless-cache \
  --serverless-cache-name your-app-valkey-serverless \
  --engine valkey \
  --serverless-cache-snapshot-retention-limit 1 \
  --daily-snapshot-time "03:00" \
  --security-group-ids sg-xxxxx \
  --subnet-ids subnet-xxxxx subnet-yyyyy subnet-zzzzz \
  --region $AWS_REGION
```

Get endpoint:
```bash
aws elasticache describe-serverless-caches \
  --serverless-cache-name your-app-valkey-serverless \
  --region $AWS_REGION \
  --query 'ServerlessCaches[0].Endpoint.Address' \
  --output text
```

### 2. Create Aurora PostgreSQL

**Via AWS Console:**
1. RDS → Create database → Aurora PostgreSQL
2. Serverless v2, minimum 0.5 ACU
3. Enable public access (if needed for migration)
4. Security group: allow K8s VPC CIDR on port 5432

**Get connection info:**
```bash
# Endpoint
aws rds describe-db-clusters \
  --db-cluster-identifier your-cluster \
  --query 'DBClusters[0].Endpoint' \
  --output text
```

### 3. Setup Redis TLS Proxy

ElastiCache Serverless requires TLS. Deploy proxy for apps without TLS support:

```bash
# Update manifests/redis-local-proxy.yaml with your Valkey endpoint
kubectl apply -f manifests/redis-local-proxy.yaml
kubectl apply -f manifests/redis-proxy-service.yaml
```

### 4. Configure Secrets

**Create Infisical secrets (or use Kubernetes secrets):**

```bash
kubectl create secret generic infisical-secrets -n posthoot \
  --from-literal=INFISICAL_API_URL=https://app.infisical.com/api \
  --from-literal=INFISICAL_CLIENT_ID=your-client-id \
  --from-literal=INFISICAL_CLIENT_SECRET=your-client-secret \
  --from-literal=INFISICAL_PROJECT_ID=your-project-id \
  --from-literal=INFISICAL_ENV=prod \
  --from-literal=REDIS_HOST=redis-proxy.posthoot.svc.cluster.local \
  --from-literal=REDIS_PORT=6379 \
  --from-literal=REDIS_URL=redis://redis-proxy.posthoot.svc.cluster.local:6379
```

**For Docker registry (if using private images):**
```bash
kubectl create secret docker-registry ghcr-secret -n posthoot \
  --docker-server=ghcr.io \
  --docker-username=your-username \
  --docker-password=your-token
```

### 5. Update Deployment Manifests

Edit `manifests/posthoot-deployment.yaml`:
```yaml
spec:
  template:
    spec:
      containers:
      - name: posthoot
        image: your-dockerhub-username/your-app:latest
```

Edit `manifests/payments-deployment.yaml` similarly.

### 6. Deploy Applications

```bash
kubectl apply -f manifests/posthoot-deployment.yaml
kubectl apply -f manifests/payments-deployment.yaml
```

### 7. Verify Deployment

```bash
# Check pods
kubectl get pods -n posthoot

# Check logs
kubectl logs -n posthoot -l app=posthoot --tail=50

# Check services
kubectl get svc -n posthoot
```

### 8. Access Grafana

```bash
# Get NLB hostname
kubectl get svc -n ingress-nginx ingress-nginx-controller \
  -o jsonpath='{.status.loadBalancer.ingress[0].hostname}'

# Get Grafana password
kubectl get secret -n monitoring kube-prometheus-grafana \
  -o jsonpath='{.data.admin-password}' | base64 -d

# Open browser: http://<NLB-HOSTNAME>
# Login: admin / <password>
```

## Configuration Guide

### Environment Variables

**Posthoot ConfigMap** (`manifests/posthoot-deployment.yaml`):
- `PORT`: Application port (default: 9001)
- `INFISICAL_PROJECT_ID`: Your Infisical project ID

**Infisical Secrets** (or K8s secrets):
- `INFISICAL_CLIENT_ID`: Infisical client ID
- `INFISICAL_CLIENT_SECRET`: Infisical client secret
- `DATABASE_URL`: PostgreSQL connection string
- `REDIS_HOST`: Redis hostname (use proxy: `redis-proxy.posthoot.svc.cluster.local`)
- `REDIS_PORT`: Redis port (6379)
- `JWT_SECRET`: JWT signing secret
- `SMTP_*`: Email configuration
- `S3_*`: S3 storage configuration

### Scaling

**Horizontal Pod Autoscaling:**
```bash
kubectl autoscale deployment posthoot -n posthoot \
  --cpu-percent=70 --min=3 --max=10
```

**Update replicas:**
```yaml
spec:
  replicas: 5  # Change in deployment YAML
```

### Resource Limits

Adjust based on load:
```yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "200m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

## Troubleshooting

### Pods Not Starting

```bash
# Check pod status
kubectl describe pod <pod-name> -n posthoot

# Check logs
kubectl logs <pod-name> -n posthoot

# Check events
kubectl get events -n posthoot --sort-by='.lastTimestamp'
```

### Database Connection Issues

```bash
# Test from pod
kubectl run -n posthoot test-db --image=postgres:15 --rm -it --restart=Never -- \
  psql -h <db-host> -U <user> -d <database>

# Check security groups allow K8s VPC CIDR on port 5432
```

### Redis Connection Issues

```bash
# Test proxy
kubectl run -n posthoot test-redis --image=redis:7-alpine --rm -it --restart=Never -- \
  redis-cli -h redis-proxy.posthoot.svc.cluster.local -p 6379 PING

# Check proxy pod
kubectl logs -n posthoot redis-local-proxy

# Test direct Valkey (requires TLS)
kubectl exec -n posthoot redis-proxy -- \
  redis-cli -h <valkey-endpoint> -p 6379 --tls --insecure PING
```

### Application Crashes

```bash
# Check readiness/liveness probes
kubectl describe pod <pod-name> -n posthoot | grep -A5 "Liveness\|Readiness"

# Check resource limits
kubectl top pods -n posthoot

# Check for OOMKilled
kubectl get pods -n posthoot -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.containerStatuses[*].lastState.terminated.reason}{"\n"}{end}'
```

### Spot Instance Termination

Handled automatically by AWS Node Termination Handler. Pods are gracefully drained before termination.

Check handler logs:
```bash
kubectl logs -n kube-system -l app.kubernetes.io/name=aws-node-termination-handler
```

## Cost Optimization

### Current Setup Costs (Monthly)
- **Compute**: ~$30-40 (spot instances)
- **ElastiCache Serverless**: $6-20 (based on usage)
- **Aurora Serverless v2**: $15-30 (0.5-1 ACU avg)
- **Data Transfer**: $5-10
- **Total**: ~$60-100/month (vs $150+ with EKS + on-demand)

### Savings
- ✅ No EKS control plane cost ($73/mo)
- ✅ Spot instances (60-80% cheaper than on-demand)
- ✅ Serverless cache (pay for use vs reserved capacity)
- ✅ Aurora Serverless v2 (scales to zero)

### Further Optimization
- Use Aurora Serverless v1 (can pause completely)
- Reduce backup retention periods
- Use S3 Intelligent-Tiering
- Enable spot instance diversification
- Right-size pod resource requests

## Monitoring & Observability

### Grafana Dashboards

1. **Posthoot Application Logs**: Per-pod log viewer
2. **Kubernetes Cluster**: Node/pod metrics
3. **Prometheus**: System metrics

### Alerts (Optional)

Create PrometheusRule for alerts:
```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: posthoot-alerts
  namespace: monitoring
spec:
  groups:
  - name: posthoot
    rules:
    - alert: PodCrashLooping
      expr: rate(kube_pod_container_status_restarts_total{namespace="posthoot"}[5m]) > 0
      for: 5m
      annotations:
        summary: "Pod {{ $labels.pod }} is crash looping"
```

## Maintenance

### Update Cluster

```bash
# Edit cluster config
kops edit cluster $CLUSTER_NAME

# Apply changes
kops update cluster $CLUSTER_NAME --yes

# Rolling update
kops rolling-update cluster $CLUSTER_NAME --yes
```

### Update Application

```bash
# Update image
kubectl set image deployment/posthoot posthoot=your-image:v2 -n posthoot

# Or apply updated manifest
kubectl apply -f manifests/posthoot-deployment.yaml

# Watch rollout
kubectl rollout status deployment/posthoot -n posthoot

# Rollback if needed
kubectl rollout undo deployment/posthoot -n posthoot
```

### Backup Database

```bash
# Aurora snapshot
aws rds create-db-cluster-snapshot \
  --db-cluster-snapshot-identifier backup-$(date +%Y%m%d) \
  --db-cluster-identifier your-cluster

# Manual dump
kubectl run -n posthoot pg-backup --image=postgres:15 --rm -it --restart=Never -- \
  pg_dump -h <db-host> -U <user> -d <database> > backup.sql
```

## Security Best Practices

1. **Secrets Management**
   - Use Infisical or AWS Secrets Manager
   - Never commit secrets to git
   - Rotate credentials regularly

2. **Network Security**
   - Use security groups to restrict access
   - Enable VPC flow logs
   - Use private subnets for databases

3. **RBAC**
   - Create service accounts with minimal permissions
   - Use namespaces for isolation
   - Audit cluster access logs

4. **Image Security**
   - Scan images for vulnerabilities
   - Use specific tags (not `latest`)
   - Pull from trusted registries

5. **TLS/Encryption**
   - Use TLS for Redis (via proxy)
   - Enable encryption at rest for RDS
   - Use SSL for PostgreSQL connections

## Cleanup

### Delete Application
```bash
kubectl delete namespace posthoot
```

### Delete Cluster
```bash
kops delete cluster $CLUSTER_NAME --yes
```

### Delete AWS Resources
```bash
# Delete ElastiCache
aws elasticache delete-serverless-cache \
  --serverless-cache-name your-app-valkey-serverless

# Delete Aurora
aws rds delete-db-cluster \
  --db-cluster-identifier your-cluster \
  --skip-final-snapshot
```

## Support & Contributing

- **Issues**: https://github.com/yourusername/posthoot/issues
- **Docs**: https://github.com/yourusername/posthoot/wiki
- **Community**: Discord/Slack link

## License

See LICENSE file in repository root.
