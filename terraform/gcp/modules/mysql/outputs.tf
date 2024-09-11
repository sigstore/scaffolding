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

output "trillian_serviceaccount" {
  description = "The email/name of the GCP service account"
  value       = google_service_account.dbuser_trillian.email
}

// Used when setting up the GKE cluster to talk to MySQL.
output "mysql_instance" {
  description = "The generated name of the Cloud SQL instance"
  value       = google_sql_database_instance.sigstore.name
}

// Full connection string for the MySQL DB>
output "mysql_connection" {
  description = "The connection string dynamically generated for storage inside the Kubernetes configmap"
  value       = format("%s:%s:%s", var.project_id, var.region, google_sql_database_instance.sigstore.name)
}

// MySQL DB username.
output "mysql_user" {
  description = "The Cloud SQL Instance User name"
  value       = google_sql_user.trillian.name
}

// MySQL DB password.
output "mysql_pass" {
  sensitive   = true
  description = "The Cloud SQL Instance Password"
  value       = google_sql_user.trillian.password
}
