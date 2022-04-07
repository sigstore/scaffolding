// Create the Rekor service account
resource "google_service_account" "rekor-sa" {
  account_id   = format("%s-rekor-sa", var.cluster_name)
  display_name = "Rekor Service Account"
  project      = var.project_id
}

resource "google_service_account_iam_member" "gke_sa_iam_member_rekor" {
  service_account_id = google_service_account.rekor-sa.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[rekor-system/rekor-server]"
  depends_on         = [google_service_account.rekor-sa]
}

resource "google_project_iam_member" "rekor_signer_verifier_member" {
  project    = var.project_id
  role       = "roles/cloudkms.signerVerifier"
  member     = "serviceAccount:${google_service_account.rekor-sa.email}"
  depends_on = [google_service_account.rekor-sa]
}

resource "google_project_iam_member" "rekor_kms_member" {
  project    = var.project_id
  role       = "roles/cloudkms.viewer"
  member     = "serviceAccount:${google_service_account.rekor-sa.email}"
  depends_on = [google_service_account.rekor-sa]
}
