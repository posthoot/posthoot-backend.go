output "cluster_name" {
  description = "GKE cluster name"
  value       = google_container_cluster.posthoot_cluster.name
}

output "cluster_endpoint" {
  description = "GKE cluster endpoint"
  value       = google_container_cluster.posthoot_cluster.endpoint
}

output "cluster_location" {
  description = "GKE cluster location"
  value       = google_container_cluster.posthoot_cluster.location
}

output "cluster_ca_certificate" {
  description = "Base64 encoded public certificate for the cluster"
  value       = google_container_cluster.posthoot_cluster.master_auth[0].cluster_ca_certificate
  sensitive   = true
}

output "get_credentials_command" {
  description = "Command to get cluster credentials"
  value       = "gcloud container clusters get-credentials ${google_container_cluster.posthoot_cluster.name} --region ${var.region} --project ${var.project_id}"
}

output "namespace" {
  description = "Kubernetes namespace"
  value       = kubernetes_namespace.posthoot.metadata[0].name
}

