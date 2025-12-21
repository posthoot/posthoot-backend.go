variable "project_id" {
  description = "The GCP project ID"
  type        = string
}

variable "region" {
  description = "The GCP region for the cluster"
  type        = string
  default     = "us-central1"
}

variable "cluster_name" {
  description = "Name of the GKE cluster"
  type        = string
  default     = "posthoot-cluster"
}

variable "namespace" {
  description = "Kubernetes namespace for the application"
  type        = string
  default     = "posthoot"
}

variable "environment" {
  description = "Environment name (dev, staging, prod)"
  type        = string
  default     = "prod"
}

# Network configuration
variable "network_name" {
  description = "Name of the VPC network"
  type        = string
  default     = "default"
}

variable "subnetwork_name" {
  description = "Name of the subnetwork"
  type        = string
  default     = "default"
}

variable "enable_private_cluster" {
  description = "Enable private cluster (nodes have private IPs only)"
  type        = bool
  default     = false
}

variable "master_ipv4_cidr_block" {
  description = "CIDR block for master nodes"
  type        = string
  default     = "172.16.0.0/28"
}

variable "master_authorized_networks" {
  description = "List of authorized networks for master access"
  type = list(object({
    cidr_block   = string
    display_name = string
  }))
  default = []
}

# Node pool configuration
variable "node_count" {
  description = "Initial number of nodes in the node pool"
  type        = number
  default     = 2
}

variable "min_node_count" {
  description = "Minimum number of nodes in the node pool"
  type        = number
  default     = 1
}

variable "max_node_count" {
  description = "Maximum number of nodes in the node pool"
  type        = number
  default     = 5
}

variable "machine_type" {
  description = "Machine type for nodes"
  type        = string
  default     = "e2-medium"
}

variable "disk_size_gb" {
  description = "Disk size in GB for nodes"
  type        = number
  default     = 50
}

variable "disk_type" {
  description = "Disk type for nodes"
  type        = string
  default     = "pd-standard"
}

variable "use_preemptible_nodes" {
  description = "Use preemptible nodes (cost savings)"
  type        = bool
  default     = false
}

variable "enable_vpa" {
  description = "Enable Vertical Pod Autoscaling"
  type        = bool
  default     = false
}

# Container Registry configuration
variable "image_registry" {
  description = "Container registry URL (e.g., ghcr.io, gcr.io, or docker.io)"
  type        = string
  default     = "ghcr.io"
}

variable "github_owner" {
  description = "GitHub username or organization name (for ghcr.io)"
  type        = string
  default     = ""
}

variable "image_name" {
  description = "Container image name (full path for ghcr.io: owner/repo-name, or just name for other registries)"
  type        = string
  default     = "posthoot"
}

variable "image_tag" {
  description = "Container image tag"
  type        = string
  default     = "latest"
}

variable "image_pull_secret_enabled" {
  description = "Enable image pull secret (required for private images)"
  type        = bool
  default     = false
}

variable "image_pull_secret_username" {
  description = "Username for image pull secret (e.g., GitHub username for ghcr.io)"
  type        = string
  default     = ""
  sensitive   = true
}

variable "image_pull_secret_password" {
  description = "Password/token for image pull secret (e.g., GitHub PAT for ghcr.io)"
  type        = string
  default     = ""
  sensitive   = true
}

# Application configuration (with defaults for basic deployment)
variable "app_replicas" {
  description = "Number of application replicas"
  type        = number
  default     = 2
}

variable "public_url" {
  description = "Public URL for the application"
  type        = string
  default     = "https://your-domain.com"
}

variable "storage_provider" {
  description = "Storage provider (local, s3)"
  type        = string
  default     = "local"
}

variable "worker_concurrency" {
  description = "Worker concurrency"
  type        = number
  default     = 5
}

variable "worker_queue_size" {
  description = "Worker queue size"
  type        = number
  default     = 100
}

variable "airley_enabled" {
  description = "Enable Airley integration"
  type        = bool
  default     = false
}

# Database configuration (with defaults - should be overridden)
variable "postgres_host" {
  description = "PostgreSQL host"
  type        = string
  default     = "your-postgres-host"
}

variable "postgres_user" {
  description = "PostgreSQL user"
  type        = string
  default     = "postgres"
}

variable "postgres_password" {
  description = "PostgreSQL password"
  type        = string
  default     = ""
  sensitive   = true
}

variable "postgres_db" {
  description = "PostgreSQL database name"
  type        = string
  default     = "posthoot"
}

# JWT configuration (with default - should be overridden)
variable "jwt_secret" {
  description = "JWT secret key"
  type        = string
  default     = "change-this-secret-key"
  sensitive   = true
}

