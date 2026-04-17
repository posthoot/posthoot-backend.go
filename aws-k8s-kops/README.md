# Kops Kubernetes Cluster with Spot Instances

Ultra-cheap production-ready Kubernetes on AWS using Kops + Spot instances.

**Cost: ~$15-25/month** (vs $150+ with EKS)

## Architecture

```
┌─────────────────────────────────────────┐
│         Kops Cluster (Multi-AZ)         │
│                                         │
│  Master (t3.micro)      $2.31/mo        │
│    └─ us-east-1a                        │
│                                         │
│  Workers (Spot)        $10-15/mo        │
│    ├─ m5.large @ $0.03/hr max          │
│    ├─ Mixed instances (5 types)        │
│    └─ Auto-spread across 3 AZs         │
│                                         │
│  Storage                $2-3/mo         │
│    └─ GP3 EBS volumes                  │
└─────────────────────────────────────────┘
```

## Quick Start

```bash
cd aws-k8s-kops

# Setup cluster (15 minutes)
chmod +x setup-kops.sh
./setup-kops.sh

# Install spot interruption handler
kubectl apply -f install-spot-handler.yaml

# Deploy apps
kubectl apply -f ../manifests/

# Verify
kubectl get nodes
kubectl get pods -A
```

## What You Get

- ✅ **Kubernetes 1.31** (latest stable)
- ✅ **Spot instances** (60-80% savings)
- ✅ **Multi-AZ** (3 zones for resilience)
- ✅ **Auto-scaling** (2-10 nodes)
- ✅ **Spot interruption handling** (2-min graceful drain)
- ✅ **Mixed instance types** (better spot availability)

## Cost Breakdown

| Resource | Type | Cost/mo |
|----------|------|---------|
| Master | t3.micro | $2.31 |
| Node 1 | m5.large spot | $5-7 |
| Node 2 | m5.large spot | $5-7 |
| EBS | GP3 (24GB) | $2-3 |
| S3 | State store | <$0.50 |
| **Total** | | **$15-20** |

Compare to EKS:
- EKS control plane: $73/mo
- Same nodes: $15/mo
- Total: **$88/mo**

**Savings: ~75%** 🎉

## Management

### Scale nodes
```bash
export KOPS_STATE_STORE=s3://posthoot-kops-state
kops edit ig nodes-us-east-1a
# Change minSize/maxSize
kops update cluster posthoot.k8s.local --yes
kops rolling-update cluster --yes
```

### Upgrade Kubernetes
```bash
kops edit cluster posthoot.k8s.local
# Change kubernetesVersion: 1.32.0
kops update cluster --yes
kops rolling-update cluster --yes
```

### Add more spot instance types
```bash
kops edit ig nodes-us-east-1a
# Add to mixedInstancesPolicy.instances:
#   - r5.large
#   - t3.large
kops update cluster --yes
```

## Spot Interruptions

AWS gives 2-minute warning before terminating spot instances.

**Handler automatically**:
1. Detects interruption notice
2. Cordons node
3. Drains pods gracefully (60s termination grace)
4. Pods reschedule on other nodes
5. Zero downtime (with PDBs)

**Monitor**:
```bash
kubectl logs -n kube-system -l app=aws-node-termination-handler -f
```

## Troubleshooting

### Cluster not ready
```bash
kops validate cluster --wait 15m
kubectl get nodes
kubectl get pods -A
```

### Spot instances not launching
```bash
# Check instance types available
aws ec2 describe-spot-price-history \
  --instance-types m5.large m5a.large c5.large \
  --product-descriptions "Linux/UNIX" \
  --max-results 10
```

### Master node crashed
```bash
# Kops auto-recovers but can force:
kops rolling-update cluster --yes --force
```

### Destroy cluster
```bash
kops delete cluster posthoot.k8s.local --yes
aws s3 rb s3://posthoot-kops-state --force
```

## Monitoring

```bash
# Cluster health
kops validate cluster

# Node status
kubectl top nodes

# Spot interruption events
kubectl get events --all-namespaces | grep -i spot

# Costs (after 1 week)
aws ce get-cost-and-usage \
  --time-period Start=2026-04-01,End=2026-04-30 \
  --granularity MONTHLY \
  --metrics BlendedCost
```

## Next Steps

1. **Deploy apps**: All existing manifests work
2. **Setup ALB**: Install AWS LB Controller
3. **Add monitoring**: Prometheus/Grafana
4. **Backups**: Velero for cluster backups
5. **CI/CD**: Update deploy workflow

## Notes

- No EKS = No $73/mo control plane fee ✅
- Spot = 60-80% cheaper than on-demand ✅
- Multi-AZ = High availability ✅
- PDBs = Zero downtime on spot interruptions ✅
- Total cost: **~$20/mo for production cluster** 🎯
