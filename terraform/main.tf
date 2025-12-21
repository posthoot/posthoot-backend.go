terraform {
  required_version = ">= 1.0"
  
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.23"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

# Enable required APIs
resource "google_project_service" "required_apis" {
  for_each = toset([
    "container.googleapis.com",
    "compute.googleapis.com",
    "iam.googleapis.com",
  ])

  project = var.project_id
  service = each.value

  disable_on_destroy = false
}

# Create GKE cluster
resource "google_container_cluster" "posthoot_cluster" {
  name     = var.cluster_name
  location = var.region

  # We can't create a cluster with no node pool defined, but we want to use
  # separately managed node pools. So we create the smallest possible default
  # node pool and immediately delete it.
  remove_default_node_pool = true
  initial_node_count       = 1

  network    = var.network_name
  subnetwork = var.subnetwork_name

  # Enable workload identity
  workload_identity_config {
    workload_pool = "${var.project_id}.svc.id.goog"
  }

  # Enable private cluster (optional, can be adjusted)
  private_cluster_config {
    enable_private_nodes    = var.enable_private_cluster
    enable_private_endpoint = false
    master_ipv4_cidr_block  = var.master_ipv4_cidr_block
  }

  # Enable network policy
  network_policy {
    enabled = true
  }

  # Enable vertical pod autoscaling (optional)
  vertical_pod_autoscaling {
    enabled = var.enable_vpa
  }

  # Enable horizontal pod autoscaling
  addons_config {
    horizontal_pod_autoscaling {
      disabled = false
    }
    http_load_balancing {
      disabled = false
    }
  }

  # Master authorized networks (optional, for better security)
  master_authorized_networks_config {
    dynamic "cidr_blocks" {
      for_each = var.master_authorized_networks
      content {
        cidr_block   = cidr_blocks.value.cidr_block
        display_name = cidr_blocks.value.display_name
      }
    }
  }

  depends_on = [
    google_project_service.required_apis,
  ]
}

# Create node pool for application workloads
resource "google_container_node_pool" "posthoot_node_pool" {
  name       = "${var.cluster_name}-node-pool"
  cluster    = google_container_cluster.posthoot_cluster.name
  location   = var.region
  node_count = var.node_count

  autoscaling {
    min_node_count = var.min_node_count
    max_node_count = var.max_node_count
  }

  management {
    auto_repair  = true
    auto_upgrade = true
  }

  node_config {
    preemptible  = var.use_preemptible_nodes
    machine_type = var.machine_type
    disk_size_gb = var.disk_size_gb
    disk_type    = var.disk_type

    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform",
      "https://www.googleapis.com/auth/compute",
      "https://www.googleapis.com/auth/devstorage.read_write",
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring",
    ]

    labels = {
      environment = var.environment
      workload    = "posthoot"
    }

    tags = ["posthoot-node"]
  }
}

# Get cluster credentials for Kubernetes provider
data "google_container_cluster" "posthoot_cluster" {
  name     = google_container_cluster.posthoot_cluster.name
  location = var.region
}

# Kubernetes provider configuration
provider "kubernetes" {
  host                   = "https://${data.google_container_cluster.posthoot_cluster.endpoint}"
  token                  = data.google_client_config.provider.access_token
  cluster_ca_certificate = base64decode(data.google_container_cluster.posthoot_cluster.master_auth[0].cluster_ca_certificate)
}

data "google_client_config" "provider" {}

# Create namespace
resource "kubernetes_namespace" "posthoot" {
  metadata {
    name = var.namespace
  }
}

