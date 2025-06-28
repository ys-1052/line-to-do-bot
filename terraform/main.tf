terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
  required_version = ">= 1.0"
}

provider "google" {
  project = var.project_id
  region  = var.region
}

variable "project_id" {
  description = "The GCP project ID"
  type        = string
}

variable "region" {
  description = "The GCP region"
  type        = string
  default     = "asia-northeast1"
}

variable "line_channel_secret" {
  description = "LINE channel secret"
  type        = string
  sensitive   = true
}

variable "line_channel_token" {
  description = "LINE channel access token"
  type        = string
  sensitive   = true
}

# Enable required APIs
resource "google_project_service" "firestore" {
  service = "firestore.googleapis.com"
}

resource "google_project_service" "run" {
  service = "run.googleapis.com"
}


resource "google_project_service" "secretmanager" {
  service = "secretmanager.googleapis.com"
}

# Create Firestore database
resource "google_firestore_database" "database" {
  project                     = var.project_id
  name                       = "(default)"
  location_id                = var.region
  type                       = "FIRESTORE_NATIVE"
  concurrency_mode           = "OPTIMISTIC"
  app_engine_integration_mode = "DISABLED"

  depends_on = [google_project_service.firestore]
}

# Secret Manager secrets
resource "google_secret_manager_secret" "line_channel_secret" {
  secret_id = "line-channel-secret"
  
  replication {
    auto {}
  }

  depends_on = [google_project_service.secretmanager]
}

resource "google_secret_manager_secret_version" "line_channel_secret" {
  secret      = google_secret_manager_secret.line_channel_secret.id
  secret_data = var.line_channel_secret
}

resource "google_secret_manager_secret" "line_channel_token" {
  secret_id = "line-channel-token"
  
  replication {
    auto {}
  }

  depends_on = [google_project_service.secretmanager]
}

resource "google_secret_manager_secret_version" "line_channel_token" {
  secret      = google_secret_manager_secret.line_channel_token.id
  secret_data = var.line_channel_token
}

# Service Account for Cloud Run
resource "google_service_account" "cloudrun_sa" {
  account_id   = "line-todo-bot-sa"
  display_name = "LINE TODO Bot Service Account"
}

# Grant necessary permissions to the service account
resource "google_project_iam_member" "firestore_user" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.cloudrun_sa.email}"
}

resource "google_project_iam_member" "secret_accessor" {
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.cloudrun_sa.email}"
}

# Cloud Run service
resource "google_cloud_run_v2_service" "line_todo_bot" {
  name     = "line-todo-bot"
  location = var.region

  template {
    service_account = google_service_account.cloudrun_sa.email
    
    containers {
      image = "asia-northeast1-docker.pkg.dev/${var.project_id}/line-todo-bot/line-todo-bot:latest"
      
      ports {
        container_port = 8080
      }
      
      env {
        name  = "GOOGLE_CLOUD_PROJECT"
        value = var.project_id
      }
      
      env {
        name = "LINE_CHANNEL_SECRET"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.line_channel_secret.secret_id
            version = "latest"
          }
        }
      }
      
      env {
        name = "LINE_CHANNEL_TOKEN"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.line_channel_token.secret_id
            version = "latest"
          }
        }
      }
      
      resources {
        limits = {
          cpu    = "1000m"
          memory = "512Mi"
        }
      }
    }

    scaling {
      min_instance_count = 0
      max_instance_count = 10
    }

    timeout = "300s"
  }

  depends_on = [google_project_service.run]
}

# Make Cloud Run service publicly accessible
resource "google_cloud_run_service_iam_binding" "run_all_users" {
  service  = google_cloud_run_v2_service.line_todo_bot.name
  location = google_cloud_run_v2_service.line_todo_bot.location
  role     = "roles/run.invoker"
  members = [
    "allUsers"
  ]
}


# Output the Cloud Run service URL
output "service_url" {
  value = google_cloud_run_v2_service.line_todo_bot.uri
  description = "The URL of the Cloud Run service"
}