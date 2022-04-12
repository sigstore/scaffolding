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
  name       = var.rekor_keyring_name
  location   = var.location
  project    = var.project_id
  depends_on = [google_project_service.service]
}

resource "google_kms_crypto_key" "rekor-key" {
  name     = var.rekor_key_name
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

resource "google_kms_key_ring" "fulcio-keyring" {
  name       = var.fulcio_keyring_name
  location   = var.location
  project    = var.project_id
  depends_on = [google_project_service.service]
}

resource "google_kms_crypto_key" "fulcio-intermediate-key" {
  name     = var.fulcio_key_name
  key_ring = google_kms_key_ring.fulcio-keyring.id
  purpose  = "ASYMMETRIC_SIGN"
  version_template {
    algorithm        = "EC_SIGN_P384_SHA384"
    protection_level = "SOFTWARE"
  }

  depends_on = [google_kms_key_ring.fulcio-keyring]
}
