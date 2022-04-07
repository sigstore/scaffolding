// Cluster policies setup.
// Provision the WIP

// Enable required services for this module
resource "google_project_service" "service" {
  for_each = toset([
    "iam.googleapis.com", // For WIP, creating service accounts and access control. roles/iam.workloadIdentityPoolAdmin, roles/iam.serviceAccountAdmin
  ])
  project = var.project_id
  service = each.key

  // Do not disable the service on destroy. On destroy, we are going to
  // destroy the project, but we need the APIs available to destroy the
  // underlying resources.
  disable_on_destroy = false
}

resource "google_iam_workload_identity_pool" "github_identity_pool" {
  project                   = var.project_id
  provider                  = google-beta
  workload_identity_pool_id = "actions-pool"
  display_name              = "GitHub Actions Pool"
  description               = "Identity pool for automated provisioning"
  depends_on                = [google_project_service.service]
}

// Provision the WIP Provider
resource "google_iam_workload_identity_pool_provider" "github_identity_provider" {
  project                            = var.project_id
  provider                           = google-beta
  workload_identity_pool_id          = google_iam_workload_identity_pool.github_identity_pool.workload_identity_pool_id
  workload_identity_pool_provider_id = "actions-provider"
  display_name                       = "Github Actions Provider"
  description                        = "OIDC identity pool provider for automated provisioning"

  attribute_mapping = {
    "google.subject"  = "assertion.sub"
    "attribute.actor" = "assertion.actor"
    "attribute.aud"   = "assertion.aud"
    // This is key!  It is used for impersonation below.
    "attribute.repository" = "assertion.repository"
  }
  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
  depends_on = [google_iam_workload_identity_pool.github_identity_pool]
}

resource "google_service_account" "github-actions-sa" {
  account_id   = format("%s-github-sa", var.cluster_name)
  display_name = "Github Actions Service Account"
  project      = var.project_id
  depends_on   = [google_project_service.service]
}

// Define the impersonation rules for this service account.
resource "google_service_account_iam_member" "allow_repository_impersonation" {
  service_account_id = google_service_account.github-actions-sa.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github_identity_pool.name}/attribute.repository/${var.github_repo}"
  depends_on = [
    google_service_account.github-actions-sa,
    google_iam_workload_identity_pool.github_identity_pool,
  ]
}
