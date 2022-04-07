// Create the Fulcio service account
resource "google_service_account" "fulcio-sa" {
  account_id   = format("%s-fulcio-sa", var.cluster_name)
  display_name = "Fulcio Service Account"
  project      = var.project_id
}

resource "google_service_account_iam_member" "gke_sa_iam_member_fulcio" {
  service_account_id = google_service_account.fulcio-sa.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[fulcio-system/fulcio-server]"
  depends_on         = [google_service_account.fulcio-sa]
}

resource "google_project_iam_member" "fulcio_member" {
  project    = var.project_id
  role       = "roles/privateca.certificateManager"
  member     = "serviceAccount:${google_service_account.fulcio-sa.email}"
  depends_on = [google_service_account.fulcio-sa]
}
