// Enable required services for this module
resource "google_project_service" "service" {
  for_each = toset([
    "cloudkms.googleapis.com", // For KMS keyring and crypto key. roles/cloudkms.admin
  ])
  project = var.project_id
  service = each.key

  // Do not disable the service on destroy. On destroy, we are going to
  // destroy the project, but we need the APIs available to destroy the
  // underlying resources.
  disable_on_destroy = false
}

resource "google_kms_key_ring" "rekor-keyring" {
  name       = var.name
  location   = var.location
  project    = var.project_id
  depends_on = [google_project_service.service]
}

resource "google_kms_crypto_key" "rekor-key" {
  name     = var.key_name
  key_ring = google_kms_key_ring.rekor-keyring.id
  purpose  = "ASYMMETRIC_SIGN"
  version_template {
    algorithm        = "EC_SIGN_P256_SHA256"
    protection_level = "SOFTWARE"
  }

  depends_on = [google_kms_key_ring.rekor-keyring]
}

resource "google_kms_key_ring_iam_member" "rekor_sa_kms_iam" {
  key_ring_id = google_kms_key_ring.rekor-keyring.id
  role        = "roles/cloudkms.viewer"
  member      = format("serviceAccount:%s-rekor-sa@%s.iam.gserviceaccount.com", var.cluster_name, var.project_id)
  depends_on  = [google_kms_key_ring.rekor-keyring]
}
