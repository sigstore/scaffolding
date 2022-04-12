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

resource "google_project_iam_member" "fulcio_kms_signer_verifier_member" {
  project    = var.project_id
  role       = "roles/cloudkms.signerVerifier"
  member     = "serviceAccount:${google_service_account.fulcio-sa.email}"
  depends_on = [google_service_account.fulcio-sa]
}

resource "google_project_iam_member" "fulcio_kms_viewer_member" {
  project    = var.project_id
  role       = "roles/cloudkms.viewer"
  member     = "serviceAccount:${google_service_account.fulcio-sa.email}"
  depends_on = [google_service_account.fulcio-sa]
}