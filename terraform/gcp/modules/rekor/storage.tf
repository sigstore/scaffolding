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

// Attestation bucket and relevant IAM
resource "google_storage_bucket" "attestation" {
  name     = var.attestation_bucket
  location = var.region
  project  = var.project_id

  storage_class               = "REGIONAL"
  uniform_bucket_level_access = true

  versioning {
    enabled = true
  }
}

// GCS Bucket 
resource "google_storage_bucket_iam_member" "rekor_gcs_member" {
  bucket     = google_storage_bucket.attestation.name
  role       = "roles/storage.objectAdmin"
  member     = "serviceAccount:${google_service_account.rekor-sa.email}"
  depends_on = [google_storage_bucket.attestation, google_service_account.rekor-sa]
}
