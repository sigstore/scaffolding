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
    "dns.googleapis.com", // For configuring DNS records
  ])
  project = var.project_id
  service = each.key

  // Do not disable the service on destroy. On destroy, we are going to
  // destroy the project, but we need the APIs available to destroy the
  // underlying resources.
  disable_on_destroy = false
}

resource "google_dns_record_set" "A_dex" {
  count = var.dns_domain_name == "" ? 0 : 1
  name  = "oauth2.${var.dns_domain_name}"
  type  = "A"
  ttl   = 60

  project      = var.project_id
  managed_zone = var.dns_zone_name

  rrdatas = [google_compute_global_address.gce_lb_ipv4.address]
}

// Create a static global IP for the external IPV4 GCE L7 load balancer
resource "google_compute_global_address" "gce_lb_ipv4" {
  name         = format("oauth2-%s-gce-ext-lb", var.cluster_name)
  address_type = "EXTERNAL"
  project      = var.project_id
}
