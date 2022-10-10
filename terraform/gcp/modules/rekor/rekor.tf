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
    "storage.googleapis.com",  // For GCS bucket. roles/storage.admin
    "cloudkms.googleapis.com", // For KMS keyring and crypto key. roles/cloudkms.admin
  ])
  project = var.project_id
  service = each.key

  // Do not disable the service on destroy. On destroy, we are going to
  // destroy the project, but we need the APIs available to destroy the
  // underlying resources.
  disable_on_destroy = false
}

// Redis for Rekor.
module "redis" {
  source = "../redis"

  region     = var.region
  project_id = var.project_id

  cluster_name = var.cluster_name

  network = var.network
}

resource "google_dns_record_set" "A_rekor" {
  name = "rekor.${var.dns_domain_name}"
  type = "A"
  ttl  = 60

  project      = var.project_id
  managed_zone = var.dns_zone_name

  rrdatas = [var.load_balancer_ipv4]
}
