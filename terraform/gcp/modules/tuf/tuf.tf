// Enable required services for this module
resource "google_project_service" "service" {
  for_each = toset([
    "storage.googleapis.com", // For GCS bucket. roles/storage.admin
  ])
  project = var.project_id
  service = each.key

  // Do not disable the service on destroy. On destroy, we are going to
  // destroy the project, but we need the APIs available to destroy the
  // underlying resources.
  disable_on_destroy = false
}

resource "google_storage_bucket" "tuf" {
  name     = var.tuf_bucket
  location = var.region
  project  = var.project_id

  storage_class               = "REGIONAL"
  uniform_bucket_level_access = true

  versioning {
    enabled = true
  }

  lifecycle_rule {
    action {
      type = "Delete"
    }
    condition {
      with_state         = "ANY"
      num_newer_versions = 10
    }
  }
  lifecycle_rule {
    action {
      type = "Delete"
    }
    condition {
      days_since_noncurrent_time = 730
    }
  }
}

resource "google_storage_bucket_iam_member" "public_tuf_member" {
  bucket = google_storage_bucket.tuf.name
  role   = "roles/storage.objectViewer"
  member = "allUsers"

  depends_on = [google_storage_bucket.tuf]
}

