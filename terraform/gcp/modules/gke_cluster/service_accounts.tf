// Create the GKE service account
resource "google_service_account" "gke-sa" {
  account_id   = format("%s-node-sa", var.cluster_name)
  display_name = "GKE Security Service Account"
  project      = var.project_id
  depends_on   = [google_project_service.service]
}

// Add the service account to the project
resource "google_project_iam_member" "service-account" {
  for_each = toset([
    "roles/logging.logWriter",
    "roles/monitoring.metricWriter",
    "roles/monitoring.viewer",
    "roles/stackdriver.resourceMetadata.writer",
  ])
  project = var.project_id
  role    = each.key
  member  = format("serviceAccount:%s", google_service_account.gke-sa.email)
}

// Create the Prometheus service account
resource "google_service_account" "prometheus-sa" {
  account_id   = format("%s-prometheus-sa", var.cluster_name)
  display_name = "Prometheus Service Account"
  project      = var.project_id
  depends_on   = [google_project_service.service]
}

resource "google_service_account_iam_member" "gke_sa_iam_member_prometheus" {
  service_account_id = google_service_account.prometheus-sa.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[prometheus/default]"
  depends_on         = [google_service_account.prometheus-sa, google_container_cluster.cluster]
}

resource "google_project_iam_member" "prometheus_member" {
  project    = var.project_id
  role       = "roles/monitoring.metricWriter"
  member     = "serviceAccount:${google_service_account.prometheus-sa.email}"
  depends_on = [google_service_account.prometheus-sa]
}
