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

resource "google_dns_record_set" "A_ctfe" {
  name = "ctfe.${var.dns_domain_name}"
  type = "A"
  ttl  = 60

  project      = var.project_id
  managed_zone = var.dns_zone_name

  rrdatas = [var.load_balancer_ipv4]
}

// For generating random suffix into the Cloud SQL instance name.
resource "random_id" "db_name_suffix" {
  byte_length = 4
}

// MYSQL for this particular CTLog
// The name of the DB Instance created should match the name of the
// shard for the CTLog.
module "mysql" {
  source = "../mysql"

  # Disable DB create/modifications if enable_ctlog_sql is false
  count = var.enable_ctlog_sql ? 1 : 0

  region     = var.region
  project_id = var.project_id

  cluster_name      = var.cluster_name
  database_version  = var.mysql_db_version
  tier              = var.mysql_tier
  availability_type = var.mysql_availability_type

  replica_zones = var.mysql_replica_zones
  replica_tier  = var.mysql_replica_tier

  network = var.network

  instance_name = var.mysql_instance_name != "" ? var.mysql_instance_name : format("%s-ctlog-mysql-%s", var.cluster_name, random_id.db_name_suffix.hex)
  db_name       = var.mysql_db_name

  ipv4_enabled              = var.mysql_ipv4_enabled
  require_ssl               = var.mysql_require_ssl
  backup_enabled            = var.mysql_backup_enabled
  binary_log_backup_enabled = var.mysql_binary_log_backup_enabled
}
