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
    "dns.googleapis.com",      // For configuring DNS records
    "cloudkms.googleapis.com", // For KMS keyring and crypto key. roles/cloudkms.admin
  ])
  project = var.project_id
  service = each.key

  // Do not disable the service on destroy. On destroy, we are going to
  // destroy the project, but we need the APIs available to destroy the
  // underlying resources.
  disable_on_destroy = false
}

resource "google_kms_key_ring" "timestamp-keyring" {
  name       = var.timestamp_keyring_name
  location   = var.kms_location
  project    = var.project_id
  depends_on = [google_project_service.service]
}

resource "google_kms_crypto_key" "timestamp-encryption-key" {
  name     = var.timestamp_encryption_key_name
  key_ring = google_kms_key_ring.timestamp-keyring.id
  # purpose defaults to symmetric encryption/decryption
  lifecycle {
    prevent_destroy = true
  }

  depends_on = [google_kms_key_ring.timestamp-keyring]
}

resource "google_kms_crypto_key" "timestamp-intermediate-ca-key" {
  name     = var.timestamp_intermediate_ca_key_name
  key_ring = google_kms_key_ring.timestamp-keyring.id
  purpose  = "ASYMMETRIC_SIGN"
  version_template {
    algorithm        = "EC_SIGN_P384_SHA384"
    protection_level = "SOFTWARE"
  }
  lifecycle {
    prevent_destroy = true
  }

  depends_on = [google_kms_key_ring.timestamp-keyring]
}