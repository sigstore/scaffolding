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

// Resources for Certificate Authority

// Enable required services for this module
resource "google_project_service" "service" {
  for_each = toset([
    "privateca.googleapis.com", // For CA and CA pool. roles/privateca.caManager
  ])
  project = var.project_id
  service = each.key

  // Do not disable the service on destroy. On destroy, we are going to
  // destroy the project, but we need the APIs available to destroy the
  // underlying resources.
  disable_on_destroy = false
}

resource "google_privateca_ca_pool" "sigstore-ca-pool" {
  name = var.ca_pool_name

  location = var.region
  tier     = "DEVOPS"
  project  = var.project_id

  publishing_options {
    publish_ca_cert = false
    publish_crl     = false
  }

  issuance_policy {
    allowed_issuance_modes {
      allow_csr_based_issuance    = true
      allow_config_based_issuance = true
    }

    identity_constraints {
      allow_subject_passthrough           = true
      allow_subject_alt_names_passthrough = true
    }

    baseline_values {
      aia_ocsp_servers = []

      ca_options {
        is_ca                       = false
        max_issuer_path_length      = 0
        non_ca                      = false
        zero_max_issuer_path_length = false
      }

      key_usage {
        base_key_usage {
          cert_sign          = false
          content_commitment = false
          crl_sign           = false
          data_encipherment  = false
          decipher_only      = false
          digital_signature  = false
          encipher_only      = false
          key_agreement      = false
          key_encipherment   = false
        }

        extended_key_usage {
          client_auth      = false
          code_signing     = false
          email_protection = false
          ocsp_signing     = false
          server_auth      = false
          time_stamping    = false
        }
      }
    }
  }

  depends_on = [google_project_service.service]
}

resource "google_privateca_certificate_authority" "sigstore-ca" {
  certificate_authority_id = var.ca_name
  location                 = var.region
  project                  = var.project_id
  pool                     = google_privateca_ca_pool.sigstore-ca-pool.name
  config {
    subject_config {
      subject {
        organization = "sigstore.dev"
        common_name  = "sigstore"
      }
    }
    x509_config {
      ca_options {
        is_ca = true
      }
      key_usage {
        base_key_usage {
          cert_sign = true
          crl_sign  = true
        }
        extended_key_usage {
        }
      }
    }
  }
  lifetime = "315360000s" # 10 years
  key_spec {
    algorithm = "EC_P384_SHA384"
  }
  type       = "SELF_SIGNED"
  depends_on = [google_privateca_ca_pool.sigstore-ca-pool]
}
