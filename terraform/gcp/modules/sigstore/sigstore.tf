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

// IAM project roles
module "project_roles" {
  source               = "../project_roles"
  project_id           = var.project_id
  iam_members_to_roles = var.iam_members_to_roles
}

// Private network
module "network" {
  source = "../network"

  region     = var.region
  project_id = var.project_id

  cluster_name = var.cluster_name

  depends_on = [
    module.project_roles
  ]
}

// Bastion
module "bastion" {
  source = "../bastion"

  project_id         = var.project_id
  region             = var.region
  zone               = var.bastion_zone
  network            = module.network.network_name
  subnetwork         = module.network.subnetwork_self_link
  tunnel_accessor_sa = var.tunnel_accessor_sa

  depends_on = [
    module.network,
    module.project_roles
  ]
}

module "tuf" {
  source = "../tuf"

  region     = var.tuf_region == "" ? var.region : var.tuf_region
  project_id = var.project_id

  tuf_bucket          = var.tuf_bucket
  tuf_bucket_member   = var.tuf_bucket_member
  gcs_logging_enabled = var.gcs_logging_enabled
  gcs_logging_bucket  = var.gcs_logging_bucket
  storage_class       = var.tuf_storage_class
  main_page_suffix    = var.tuf_main_page_suffix

  tuf_service_account_name = var.tuf_service_account_name

  tuf_keyring_name = var.tuf_keyring_name
  tuf_key_name     = var.tuf_key_name
  kms_location     = var.tuf_kms_location

  depends_on = [
    module.project_roles
  ]
}

// Monitoring
module "monitoring" {
  source = "../monitoring"

  // Disable module entirely if monitoring
  // is disabled
  count = var.monitoring.enabled ? 1 : 0

  project_id               = var.project_id
  project_number           = var.project_number
  cluster_location         = module.gke-cluster.cluster_location
  cluster_name             = var.cluster_name
  ca_pool_name             = var.ca_pool_name
  fulcio_url               = var.monitoring.fulcio_url
  rekor_url                = var.monitoring.rekor_url
  dex_url                  = var.monitoring.dex_url
  ctlog_url                = var.monitoring.ctlog_url
  notification_channel_ids = var.monitoring.notification_channel_ids
  create_slos              = var.create_slos

  depends_on = [
    module.gke-cluster,
    module.project_roles
  ]
}

resource "google_compute_firewall" "bastion-egress" {
  // Egress to Kubernetes API is the only allowed traffic
  name      = "bastion-egress"
  network   = module.network.network_name
  direction = "EGRESS"

  destination_ranges = ["${module.gke-cluster.cluster_endpoint}/32"]

  allow {
    protocol = "tcp"
    ports    = ["443"]
  }

  target_tags = ["bastion"]

  depends_on = [
    module.network,
    module.gke-cluster,
    module.project_roles
  ]
}

# GKE cluster setup.
module "gke-cluster" {
  source = "../gke_cluster"

  region     = var.region
  project_id = var.project_id

  cluster_name = var.cluster_name

  network                       = module.network.network_self_link
  subnetwork                    = module.network.subnetwork_self_link
  cluster_secondary_range_name  = module.network.secondary_ip_range.0.range_name
  services_secondary_range_name = module.network.secondary_ip_range.1.range_name
  cluster_network_tag           = var.cluster_network_tag

  initial_node_count   = var.initial_node_count
  autoscaling_min_node = var.autoscaling_min_node
  autoscaling_max_node = var.autoscaling_max_node

  resource_limits_resource_cpu_max = var.gke_autoscaling_resource_limits_resource_cpu_max
  resource_limits_resource_mem_max = var.gke_autoscaling_resource_limits_resource_mem_max

  bastion_ip_address = module.bastion.ip_address

  monitoring_components = var.cluster_monitoring_components

  security_group = var.gke_cluster_security_group

  oauth_scopes = var.gke_oauth_scopes

  depends_on = [
    module.network,
    module.bastion,
    module.project_roles
  ]
}

// MYSQL. This is the original DB that was used for both Rekor and CTLog.
// Newer versions of CTLog create their own database instance, so there's
// one database instance to a single ctlog shard.
module "mysql" {
  source = "../mysql"

  region     = var.region
  project_id = var.project_id

  cluster_name      = var.cluster_name
  database_version  = var.mysql_db_version
  tier              = var.mysql_tier
  availability_type = var.mysql_availability_type
  collation         = var.mysql_collation

  replica_zones = var.mysql_replica_zones
  replica_tier  = var.mysql_replica_tier

  network = module.network.network_self_link

  instance_name = var.mysql_instance_name
  db_name       = var.mysql_db_name

  ipv4_enabled              = var.mysql_ipv4_enabled
  require_ssl               = var.mysql_require_ssl
  backup_enabled            = var.mysql_backup_enabled
  binary_log_backup_enabled = var.mysql_binary_log_backup_enabled


  depends_on = [
    module.network,
    module.gke-cluster,
    module.project_roles
  ]
}


// Rekor
module "rekor" {
  source = "../rekor"

  region       = var.region
  project_id   = var.project_id
  cluster_name = var.cluster_name

  // Redis
  network = module.network.network_self_link

  // KMS
  rekor_keyring_name = var.rekor_keyring_name
  rekor_key_name     = var.rekor_key_name
  kms_location       = "global"

  // Storage
  attestation_bucket  = var.attestation_bucket
  attestation_region  = var.attestation_region == "" ? var.region : var.attestation_region
  gcs_logging_enabled = var.gcs_logging_enabled
  gcs_logging_bucket  = var.gcs_logging_bucket
  storage_class       = var.attestation_storage_class

  dns_zone_name   = var.dns_zone_name
  dns_domain_name = var.dns_domain_name

  new_entry_pubsub_consumers = var.rekor_new_entry_pubsub_consumers

  redis_cluster_memory_size_gb = var.redis_cluster_memory_size_gb

  depends_on = [
    module.network,
    module.gke-cluster,
    module.project_roles
  ]
}

// Fulcio
module "fulcio" {
  source = "../fulcio"

  region       = var.region
  project_id   = var.project_id
  cluster_name = var.cluster_name

  // Certificate authority
  ca_pool_name = var.ca_pool_name
  ca_name      = var.ca_name

  // KMS
  fulcio_keyring_name = var.fulcio_keyring_name
  fulcio_key_name     = var.fulcio_intermediate_key_name

  dns_zone_name   = var.dns_zone_name
  dns_domain_name = var.dns_domain_name

  depends_on = [
    module.gke-cluster,
    module.network,
    module.project_roles
  ]
}

module "timestamp" {
  source = "../timestamp"

  region       = var.region
  project_id   = var.project_id
  cluster_name = var.cluster_name

  // Disable module entirely if timestamp
  // is disabled
  count = var.timestamp.enabled ? 1 : 0

  // KMS
  timestamp_keyring_name             = var.timestamp_keyring_name
  timestamp_encryption_key_name      = var.timestamp_encryption_key_name
  timestamp_intermediate_ca_key_name = var.timestamp_intermediate_ca_key_name

  dns_zone_name   = var.dns_zone_name
  dns_domain_name = var.dns_domain_name

  depends_on = [
    module.gke-cluster,
    module.network,
    module.project_roles
  ]
}

// Audit
module "audit" {
  source     = "../audit"
  project_id = var.project_id
}

// OSLogin configuration
module "oslogin" {
  source     = "../oslogin"
  project_id = var.project_id

  // Disable module entirely if oslogin is disabled
  count = var.oslogin.enabled ? 1 : 0

  oslogin = var.oslogin

  // Grant OSLogin access to the bastion instance to the GHA
  // SA for terraform access and to tunnel accessors.
  instance_os_login_members = {
    bastion = {
      instance_name = module.bastion.name
      zone          = module.bastion.zone
      members       = var.tunnel_accessor_sa
    }
  }
  depends_on = [
    module.bastion,
    module.project_roles
  ]
}

// ctlog. This was the original (pre-ga) ctlog that shared the DB instance
// with Rekor.
module "ctlog" {
  source = "../ctlog"

  project_id   = var.project_id
  cluster_name = var.cluster_name

  dns_zone_name   = var.dns_zone_name
  dns_domain_name = var.dns_domain_name

  depends_on = [
    module.gke-cluster,
    module.network,
    module.project_roles
  ]
}

// ctlog-shards. This will create CTLog shard that has its own Cloud SQL
// instance for each shard
module "ctlog_shards" {
  source = "../mysql-shard"

  for_each = var.ctlog_shards

  instance_name = each.value["instance_name"] != "" ? each.value["instance_name"] : format("%s-ctlog-%s", var.cluster_name, each.key)

  project_id = var.project_id
  region     = var.region

  cluster_name = var.cluster_name

  database_version = each.value["mysql_db_version"]
  tier             = each.value["mysql_tier"] != "" ? each.value["mysql_tier"] : var.mysql_tier

  replica_zones = var.mysql_replica_zones
  replica_tier  = var.mysql_replica_tier

  // We want to use consistent password across mysql DB instances, because
  // this is access only at the DB level and access to the DB instance is gated
  // by the IAM as well as private network.
  password = module.mysql.mysql_pass

  network = module.network.network_self_link

  db_name = var.ctlog_mysql_db_name

  ipv4_enabled              = var.mysql_ipv4_enabled
  require_ssl               = var.mysql_require_ssl
  backup_enabled            = var.mysql_backup_enabled
  binary_log_backup_enabled = var.mysql_binary_log_backup_enabled
  collation                 = var.mysql_collation


  depends_on = [
    module.gke-cluster,
    module.network,
    // Need to make sure we have the necessary network, service accounts, and
    // services.
    module.mysql
  ]
}

// standalone-mysql. This will create a MySQL database that is not part of
// something else. This is used to bring a database up with the appropriate
// permissions / connections so that it can be used then by manually wiring
// it to places where it's needed. This was initially created to bring up
// a different version of a database that we needed to migrate to.

module "standalone_mysqls" {
  source = "../mysql-shard"

  for_each = toset(var.standalone_mysqls)

  instance_name = format("%s-standalone-%s", var.cluster_name, each.key)

  project_id = var.project_id
  region     = var.region

  cluster_name = var.cluster_name
  // NB: This is commented out so that we pick up the defaults
  // for the particular environment consistently.
  //mysql_database_version  = var.mysql_db_version

  tier = var.standalone_mysql_tier

  replica_zones = var.mysql_replica_zones
  replica_tier  = var.mysql_replica_tier

  // We want to use consistent password across mysql DB instances, because
  // this is access only at the DB level and access to the DB instance is gated
  // by the IAM as well as private network.
  password = module.mysql.mysql_pass

  network = module.network.network_self_link

  db_name = var.mysql_db_name

  ipv4_enabled              = var.mysql_ipv4_enabled
  require_ssl               = var.standalone_mysql_ssl
  backup_enabled            = var.mysql_backup_enabled
  binary_log_backup_enabled = var.mysql_binary_log_backup_enabled
  collation                 = var.mysql_collation


  depends_on = [
    module.gke-cluster,
    module.network,
    // Need to make sure we have the necessary network, service accounts, and
    // services.
    module.mysql
  ]
}

// dex
module "dex" {
  source = "../dex"

  project_id = var.project_id

  cluster_name = var.cluster_name

  dns_zone_name   = var.dns_zone_name
  dns_domain_name = var.dns_domain_name

  depends_on = [
    module.gke-cluster,
    module.network,
    module.project_roles
  ]
}
