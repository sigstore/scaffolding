// Enable required services for this module
resource "google_project_service" "service" {
  for_each = toset([
    "monitoring.googleapis.com", // For monitoring alerts.
  ])
  project = var.project_id
  service = each.key

  // Do not disable the service on destroy. On destroy, we are going to
  // destroy the project, but we need the APIs available to destroy the
  // underlying resources.
  disable_on_destroy = false
}

// Pull in submodules for rekor, fulcio, and dex

// Rekor
module "rekor" {
  source = "./rekor"

  project_id              = var.project_id
  notification_channel_id = var.notification_channel_id

  depends_on = [
    google_project_service.service
  ]
}

// Fulcio
module "fulcio" {
  source = "./fulcio"

  project_id              = var.project_id
  notification_channel_id = var.notification_channel_id

  depends_on = [
    google_project_service.service
  ]
}

// Dex
module "dex" {
  source = "./dex"

  project_id              = var.project_id
  notification_channel_id = var.notification_channel_id

  depends_on = [
    google_project_service.service
  ]
}

