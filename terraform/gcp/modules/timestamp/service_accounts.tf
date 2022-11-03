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

// Create the Timestamp Authority service account
resource "google_service_account" "timestamp-sa" {
  account_id   = format("%s-timestamp-sa", var.cluster_name)
  display_name = "Timestamp Authority Service Account"
  project      = var.project_id
}

resource "google_service_account_iam_member" "gke_sa_iam_member_timestamp" {
  service_account_id = google_service_account.timestamp-sa.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[timestamp-system/timestamp-server]"
  depends_on         = [google_service_account.timestamp-sa]
}

resource "google_project_iam_member" "timestamp_kms_signer_verifier_member" {
  project    = var.project_id
  role       = "roles/cloudkms.signerVerifier"
  member     = "serviceAccount:${google_service_account.timestamp-sa.email}"
  depends_on = [google_service_account.timestamp-sa]
}

// Decrypt encrypted Tink keyset to get signing key
resource "google_project_iam_member" "timestamp_kms_decrypter_member" {
  project    = var.project_id
  role       = "roles/cloudkms.cryptoKeyDecrypter"
  member     = "serviceAccount:${google_service_account.timestamp-sa.email}"
  depends_on = [google_service_account.timestamp-sa]
}

resource "google_project_iam_member" "timestamp_kms_viewer_member" {
  project    = var.project_id
  role       = "roles/cloudkms.viewer"
  member     = "serviceAccount:${google_service_account.timestamp-sa.email}"
  depends_on = [google_service_account.timestamp-sa]
}