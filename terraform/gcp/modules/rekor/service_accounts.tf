/**
 * Copyright 2022 The Sigstore Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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

resource "google_project_iam_member" "rekor_profiler_agent" {
  project    = var.project_id
  role       = "roles/cloudprofiler.agent"
  member     = "serviceAccount:${google_service_account.rekor-sa.email}"
  depends_on = [google_service_account.rekor-sa]
}

resource "google_service_account_iam_member" "gke_sa_iam_member_rekor_server" {
  service_account_id = google_service_account.rekor-sa.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[rekor-system/rekor-server]"
  depends_on         = [google_service_account.rekor-sa]
}

resource "google_project_iam_member" "db_admin_member_rekor" {
  project    = var.project_id
  role       = "roles/cloudsql.client"
  member     = "serviceAccount:${google_service_account.rekor-sa.email}"
  depends_on = [google_service_account.rekor-sa]
}

resource "google_project_iam_member" "logserver_iam" {
  # // Give rekor permission to export metrics to Stackdriver
  for_each = toset([
    "roles/logging.logWriter",
    "roles/monitoring.metricWriter",
    "roles/stackdriver.resourceMetadata.writer",
    "roles/cloudtrace.agent"
  ])
  project    = var.project_id
  role       = each.key
  member     = "serviceAccount:${google_service_account.rekor-sa.email}"
  depends_on = [google_service_account.rekor-sa]
}
