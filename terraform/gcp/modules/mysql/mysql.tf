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

# MySQL setup
# Forked from https://github.com/GoogleCloudPlatform/gke-private-cluster-demo/blob/master/terraform/postgres.tf

// Enable required services for this module
resource "google_project_service" "service" {
  for_each = toset([
    "cloudresourcemanager.googleapis.com", // For IAM bindings. roles/resourcemanager.projectIamAdmin
    "compute.googleapis.com",              // For compute global address. roles/compute.networkAdmin
    "iam.googleapis.com",                  // For creating service accounts and access control. roles/iam.serviceAccountAdmin
    "secretmanager.googleapis.com",        // For Secrets. roles/secretmanager.admin
    "servicenetworking.googleapis.com",    // For service networking connection. roles/servicenetworking.networksAdmin
    "sqladmin.googleapis.com",             // For Cloud SQL. roles/cloudsql.admin
  ])
  project = var.project_id
  service = each.key

  // Do not disable the service on destroy. On destroy, we are going to
  // destroy the project, but we need the APIs available to destroy the
  // underlying resources.
  disable_on_destroy = false
}

// Access to private cluster
resource "google_compute_global_address" "private_ip_address" {
  name          = format("%s-priv-ip", var.cluster_name)
  project       = var.project_id
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = 16
  network       = var.network
  depends_on    = [google_project_service.service]
}

resource "google_service_networking_connection" "private_vpc_connection" {
  network                 = var.network
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.private_ip_address.name]
  depends_on              = [google_compute_global_address.private_ip_address]
}

// Create the Google SA
resource "google_service_account" "dbuser_trillian" {
  account_id   = format("%s-mysql-sa", var.cluster_name)
  display_name = "Trillian SA"
  project      = var.project_id
  depends_on   = [google_project_service.service]
}

// Attach cloudsql access permissions to the Google SA.
resource "google_project_iam_member" "db_admin_member_trillian" {
  project    = var.project_id
  role       = "roles/cloudsql.client"
  member     = "serviceAccount:${google_service_account.dbuser_trillian.email}"
  depends_on = [google_service_account.dbuser_trillian]
}

resource "google_service_account_iam_member" "gke_sa_iam_member_trillian_logserver" {
  service_account_id = google_service_account.dbuser_trillian.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[trillian-system/trillian-logserver]"
  depends_on         = [google_service_account.dbuser_trillian]
}

resource "google_project_iam_member" "logserver_iam" {
  # // Give trillian logserver permission to export metrics to Stackdriver
  for_each = toset([
    "roles/logging.logWriter",
    "roles/monitoring.metricWriter",
    "roles/stackdriver.resourceMetadata.writer",
    "roles/cloudtrace.agent"
  ])
  project    = var.project_id
  role       = each.key
  member     = "serviceAccount:${google_service_account.dbuser_trillian.email}"
  depends_on = [google_service_account.dbuser_trillian]
}

resource "google_service_account_iam_member" "gke_sa_iam_member_trillian_logsigner" {
  service_account_id = google_service_account.dbuser_trillian.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[trillian-system/trillian-logsigner]"
  depends_on         = [google_service_account.dbuser_trillian]
}

resource "random_id" "db_name_suffix" {
  byte_length = 4
}

resource "google_sql_database_instance" "sigstore" {
  project          = var.project_id
  name             = var.instance_name != "" ? var.instance_name : format("%s-mysql-%s", var.cluster_name, random_id.db_name_suffix.hex)
  database_version = var.database_version
  region           = var.region

  # Set to false to delete this database
  deletion_protection = var.deletion_protection

  depends_on = [google_service_networking_connection.private_vpc_connection]

  settings {
    tier              = var.tier
    activation_policy = "ALWAYS"
    availability_type = var.availability_type

    ip_configuration {
      ipv4_enabled    = var.ipv4_enabled
      private_network = var.network
      require_ssl     = var.require_ssl
    }

    database_flags {
      name  = "cloudsql_iam_authentication"
      value = "on"
    }

    backup_configuration {
      enabled            = var.backup_enabled
      binary_log_enabled = var.binary_log_backup_enabled
    }
  }

  timeouts {
    create = "10m"
    update = "10m"
    delete = "10m"
  }
}

moved {
  from = google_sql_database_instance.trillian
  to   = google_sql_database_instance.sigstore
}

resource "google_sql_database_instance" "read_replica" {
  for_each = toset(var.replica_zones)

  name                 = "${google_sql_database_instance.sigstore.name}-replica-${each.key}"
  master_instance_name = google_sql_database_instance.sigstore.name
  region               = var.region
  database_version     = var.database_version

  replica_configuration {
    failover_target = false
  }

  settings {
    tier              = var.replica_tier
    availability_type = "ZONAL"

    ip_configuration {
      ipv4_enabled    = var.ipv4_enabled
      private_network = var.network
      require_ssl     = var.require_ssl
    }

    database_flags {
      name  = "cloudsql_iam_authentication"
      value = "on"
    }
  }
}

resource "google_sql_database" "trillian" {
  name       = var.db_name
  project    = var.project_id
  instance   = google_sql_database_instance.sigstore.name
  collation  = "utf8_general_ci"
  depends_on = [google_sql_database_instance.sigstore]
}

resource "google_sql_database" "searchindexes" {
  name       = var.index_db_name
  project    = var.project_id
  instance   = google_sql_database_instance.sigstore.name
  collation  = "utf8_general_ci"
  depends_on = [google_sql_database_instance.sigstore]
}

resource "random_id" "user-password" {
  keepers = {
    name = google_sql_database_instance.sigstore.name
  }

  byte_length = 8
  depends_on  = [google_sql_database_instance.sigstore]
}

resource "google_sql_user" "trillian" {
  name       = "trillian"
  project    = var.project_id
  instance   = google_sql_database_instance.sigstore.name
  password   = random_id.user-password.hex
  host       = "%"
  depends_on = [google_sql_database_instance.sigstore]
}

resource "google_secret_manager_secret" "mysql-password" {
  secret_id = "mysql-password"

  replication {
    auto {}
  }
  depends_on = [google_project_service.service]
}

resource "google_secret_manager_secret_version" "mysql-password" {
  secret      = google_secret_manager_secret.mysql-password.id
  secret_data = google_sql_user.trillian.password
  depends_on  = [google_secret_manager_secret.mysql-password]
}

resource "google_secret_manager_secret" "mysql-user" {
  secret_id = "mysql-user"

  replication {
    auto {}
  }
  depends_on = [google_project_service.service]
}

resource "google_secret_manager_secret_version" "mysql-user" {
  secret      = google_secret_manager_secret.mysql-user.id
  secret_data = google_sql_user.trillian.name
}

resource "google_secret_manager_secret" "mysql-database" {
  secret_id = "mysql-database"

  replication {
    auto {}
  }
  depends_on = [google_project_service.service]
}

resource "google_secret_manager_secret_version" "mysql-database" {
  secret      = google_secret_manager_secret.mysql-database.id
  secret_data = google_sql_database.trillian.name
}
