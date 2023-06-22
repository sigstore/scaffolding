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
    "compute.googleapis.com",           // For compute global address. roles/compute.networkAdmin
    "redis.googleapis.com",             // For Redis memorystore.
    "servicenetworking.googleapis.com", // For service networking connection. roles/servicenetworking.networksAdmin
  ])
  project = var.project_id
  service = each.key

  // Do not disable the service on destroy. On destroy, we are going to
  // destroy the project, but we need the APIs available to destroy the
  // underlying resources.
  disable_on_destroy = false
}


// Access to private cluster
resource "google_compute_global_address" "service_range" {
  name          = format("%s-priv-ip", var.cluster_name)
  project       = var.project_id
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = 16
  network       = var.network
}

resource "google_service_networking_connection" "private_service_connection" {
  network                 = var.network
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.service_range.name]
}

data "google_compute_zones" "available" {
  // All available AZ in our region
  region = var.region
}

resource "random_shuffle" "redis_az" {
  // Randomly select two AZ from our region for the redis
  input        = data.google_compute_zones.available.names
  result_count = 2

  lifecycle {
    ignore_changes = all
  }
}

resource "google_redis_instance" "index" {
  display_name   = "Rekor Index Instance"
  name           = "rekor-index"
  tier           = "STANDARD_HA"
  memory_size_gb = var.memory_size_gb
  redis_version  = "REDIS_6_X"

  region                  = var.region // Used for naming, location determined by location_id
  location_id             = random_shuffle.redis_az.result[0]
  alternative_location_id = random_shuffle.redis_az.result[1]

  transit_encryption_mode = "DISABLED" // Consider enabling when Rekor is updated to support TLS with Redis client.

  authorized_network = var.network
  connect_mode       = "PRIVATE_SERVICE_ACCESS"


  depends_on = [google_service_networking_connection.private_service_connection]

}
