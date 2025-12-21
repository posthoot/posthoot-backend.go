# Deploy Redis
resource "kubernetes_deployment" "redis" {
  metadata {
    name      = "redis"
    namespace = kubernetes_namespace.posthoot.metadata[0].name
    labels = {
      app = "redis"
    }
  }

  spec {
    replicas = 1

    selector {
      matchLabels = {
        app = "redis"
      }
    }

    template {
      metadata {
        labels = {
          app = "redis"
        }
      }

      spec {
        container {
          name  = "redis"
          image = "redis:7-alpine"

          port {
            container_port = 6379
            name           = "redis"
          }

          command = ["redis-server", "--appendonly", "yes"]

          resources {
            requests = {
              memory = "128Mi"
              cpu    = "100m"
            }
            limits = {
              memory = "256Mi"
              cpu    = "200m"
            }
          }

          volume_mount {
            name       = "redis-data"
            mount_path = "/data"
          }

          liveness_probe {
            exec {
              command = ["redis-cli", "ping"]
            }
            initial_delay_seconds = 30
            period_seconds        = 10
            timeout_seconds       = 5
          }

          readiness_probe {
            exec {
              command = ["redis-cli", "ping"]
            }
            initial_delay_seconds = 5
            period_seconds        = 5
            timeout_seconds       = 3
          }
        }

        volume {
          name = "redis-data"
          empty_dir {}
        }
      }
    }
  }
}

resource "kubernetes_service" "redis" {
  metadata {
    name      = "redis"
    namespace = kubernetes_namespace.posthoot.metadata[0].name
    labels = {
      app = "redis"
    }
  }

  spec {
    type = "ClusterIP"

    port {
      port        = 6379
      target_port = 6379
      protocol    = "TCP"
      name        = "redis"
    }

    selector = {
      app = "redis"
    }
  }
}

# Deploy Application
resource "kubernetes_deployment" "posthoot" {
  metadata {
    name      = "posthoot"
    namespace = kubernetes_namespace.posthoot.metadata[0].name
    labels = {
      app = "posthoot"
    }
  }

  spec {
    replicas = var.app_replicas

    selector {
      matchLabels = {
        app = "posthoot"
      }
    }

    template {
      metadata {
        labels = {
          app = "posthoot"
        }
      }

      spec {
        container {
          name  = "posthoot"
          image = var.image_registry == "ghcr.io" && var.github_owner != "" ? "${var.image_registry}/${var.github_owner}/${var.image_name}:${var.image_tag}" : "${var.image_registry}/${var.project_id}/${var.image_name}:${var.image_tag}"
          image_pull_policy = "Always"

          port {
            container_port = 9001
            name           = "http"
          }

          env {
            name  = "SERVER_HOST"
            value = "0.0.0.0"
          }

          env {
            name  = "SERVER_PORT"
            value = "9001"
          }

          env {
            name  = "REDIS_HOST"
            value = "redis"
          }

          env {
            name  = "REDIS_PORT"
            value = "6379"
          }

          env {
            name = "POSTGRES_HOST"
            value_from {
              secret_key_ref {
                name = kubernetes_secret.posthoot_secrets.metadata[0].name
                key  = "postgres-host"
              }
            }
          }

          env {
            name  = "POSTGRES_PORT"
            value = "5432"
          }

          env {
            name = "POSTGRES_USER"
            value_from {
              secret_key_ref {
                name = kubernetes_secret.posthoot_secrets.metadata[0].name
                key  = "postgres-user"
              }
            }
          }

          env {
            name = "POSTGRES_PASSWORD"
            value_from {
              secret_key_ref {
                name = kubernetes_secret.posthoot_secrets.metadata[0].name
                key  = "postgres-password"
              }
            }
          }

          env {
            name = "POSTGRES_DB"
            value_from {
              secret_key_ref {
                name = kubernetes_secret.posthoot_secrets.metadata[0].name
                key  = "postgres-db"
              }
            }
          }

          env {
            name  = "POSTGRES_SSLMODE"
            value = "require"
          }

          env {
            name = "JWT_SECRET"
            value_from {
              secret_key_ref {
                name = kubernetes_secret.posthoot_secrets.metadata[0].name
                key  = "jwt-secret"
              }
            }
          }

          env {
            name = "PUBLIC_URL"
            value_from {
              config_map_key_ref {
                name = kubernetes_config_map.posthoot_config.metadata[0].name
                key  = "public-url"
              }
            }
          }

          env {
            name = "STORAGE_PROVIDER"
            value_from {
              config_map_key_ref {
                name = kubernetes_config_map.posthoot_config.metadata[0].name
                key  = "storage-provider"
              }
            }
          }

          env {
            name  = "STORAGE_BASE_PATH"
            value = "/app/storage"
          }

          env {
            name = "WORKER_CONCURRENCY"
            value_from {
              config_map_key_ref {
                name = kubernetes_config_map.posthoot_config.metadata[0].name
                key  = "worker-concurrency"
              }
            }
          }

          env {
            name = "WORKER_QUEUE_SIZE"
            value_from {
              config_map_key_ref {
                name = kubernetes_config_map.posthoot_config.metadata[0].name
                key  = "worker-queue-size"
              }
            }
          }

          env {
            name = "AIRLEY_ENABLED"
            value_from {
              config_map_key_ref {
                name = kubernetes_config_map.posthoot_config.metadata[0].name
                key  = "airley-enabled"
              }
            }
          }

          resources {
            requests = {
              memory = "256Mi"
              cpu    = "200m"
            }
            limits = {
              memory = "512Mi"
              cpu    = "500m"
            }
          }

          liveness_probe {
            http_get {
              path = "/health"
              port = 9001
            }
            initial_delay_seconds = 30
            period_seconds        = 10
            timeout_seconds       = 5
          }

          readiness_probe {
            http_get {
              path = "/health"
              port = 9001
            }
            initial_delay_seconds = 10
            period_seconds        = 5
            timeout_seconds       = 3
          }
        }

        dynamic "image_pull_secrets" {
          for_each = var.image_pull_secret_enabled ? [1] : []
          content {
            name = kubernetes_secret.registry_secret[0].metadata[0].name
          }
        }
      }
    }
  }
}

resource "kubernetes_service" "posthoot" {
  metadata {
    name      = "posthoot"
    namespace = kubernetes_namespace.posthoot.metadata[0].name
    labels = {
      app = "posthoot"
    }
  }

  spec {
    type = "LoadBalancer"

    port {
      port        = 80
      target_port = 9001
      protocol    = "TCP"
      name        = "http"
    }

    selector = {
      app = "posthoot"
    }
  }
}

# ConfigMap for application configuration
resource "kubernetes_config_map" "posthoot_config" {
  metadata {
    name      = "posthoot-config"
    namespace = kubernetes_namespace.posthoot.metadata[0].name
  }

  data = {
    public-url         = var.public_url
    storage-provider   = var.storage_provider
    worker-concurrency = tostring(var.worker_concurrency)
    worker-queue-size  = tostring(var.worker_queue_size)
    airley-enabled     = tostring(var.airley_enabled)
  }
}

# Secret for sensitive configuration
resource "kubernetes_secret" "posthoot_secrets" {
  metadata {
    name      = "posthoot-secrets"
    namespace = kubernetes_namespace.posthoot.metadata[0].name
  }

  data = {
    "postgres-host"     = base64encode(var.postgres_host)
    "postgres-user"     = base64encode(var.postgres_user)
    "postgres-password" = base64encode(var.postgres_password)
    "postgres-db"       = base64encode(var.postgres_db)
    "jwt-secret"        = base64encode(var.jwt_secret)
  }
}

# Secret for container registry authentication (optional, only for private images)
resource "kubernetes_secret" "registry_secret" {
  count = var.image_pull_secret_enabled ? 1 : 0

  metadata {
    name      = "registry-pull-secret"
    namespace = kubernetes_namespace.posthoot.metadata[0].name
  }

  type = "kubernetes.io/dockerconfigjson"

  data = {
    ".dockerconfigjson" = jsonencode({
      auths = {
        "${var.image_registry}" = {
          "username" = var.image_registry == "ghcr.io" ? var.image_pull_secret_username : "_json_key"
          "password" = var.image_registry == "ghcr.io" ? var.image_pull_secret_password : var.image_pull_secret_password
          "auth"     = base64encode("${var.image_registry == "ghcr.io" ? var.image_pull_secret_username : "_json_key"}:${var.image_pull_secret_password}")
        }
      }
    })
  }
}

