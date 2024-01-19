/**
 * Copyright 2024 The Sigstore Authors
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

resource "google_kms_key_ring" "tuf-keyring" {
  name     = var.tuf_keyring_name
  location = var.kms_location
  project  = var.project_id
}

resource "google_kms_crypto_key" "tuf-key" {
  name     = var.tuf_key_name
  key_ring = google_kms_key_ring.tuf-keyring.id
  purpose  = "ASYMMETRIC_SIGN"
  version_template {
    algorithm        = "EC_SIGN_P256_SHA256"
    protection_level = "SOFTWARE"
  }
  lifecycle {
    prevent_destroy = true
  }
  depends_on = [google_kms_key_ring.tuf-keyring]
}

resource "google_kms_crypto_key_version" "tuf-key-version" {
  crypto_key = google_kms_crypto_key.tuf-key.id
  depends_on = [google_kms_crypto_key.tuf-key]
}

resource "google_kms_key_ring_iam_member" "tuf-sa-key-iam" {
  key_ring_id = google_kms_key_ring.tuf-keyring.id
  role        = "roles/cloudkms.signerVerifier"
  member      = format("serviceAccount:%s@%s.iam.gserviceaccount.com", var.tuf_service_account_name, var.project_id)
  depends_on  = [google_kms_key_ring.tuf-keyring, google_service_account.tuf-sa]
}

resource "google_kms_key_ring_iam_member" "tuf-key-iam-viewers" {
  for_each = toset(var.tuf_key_viewers)

  key_ring_id = google_kms_key_ring.tuf-keyring.id
  role        = "roles/cloudkms.publicKeyViewer"
  member      = each.key
  depends_on  = [google_kms_key_ring.tuf-keyring]
}
