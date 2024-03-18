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

// Attestation bucket and relevant IAM
resource "google_storage_bucket" "attestation" {
  count    = var.enable_attestations ? 1 : 0
  name     = var.attestation_bucket
  location = var.attestation_region == "" ? var.region : var.attestation_region
  project  = var.project_id

  storage_class               = var.storage_class
  uniform_bucket_level_access = true

  versioning {
    enabled = true
  }

  dynamic "logging" {
    for_each = var.gcs_logging_enabled ? [1] : []
    content {
      log_bucket = var.gcs_logging_bucket
    }
  }
}

// GCS Bucket 
resource "google_storage_bucket_iam_member" "rekor_gcs_member" {
  count      = var.enable_attestations ? 1 : 0
  bucket     = google_storage_bucket.attestation[count.index].name
  role       = "roles/storage.objectAdmin"
  member     = "serviceAccount:${google_service_account.rekor-sa.email}"
  depends_on = [google_storage_bucket.attestation, google_service_account.rekor-sa]
}
