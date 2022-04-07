output "trillian_serviceaccount" {
  description = "The email/name of the GCP service account"
  value       = google_service_account.dbuser_trillian.email
}

// Used when setting up the GKE cluster to talk to MySQL.
output "mysql_instance" {
  description = "The generated name of the Cloud SQL instance"
  value       = google_sql_database_instance.trillian.name
}

// Full connection string for the MySQL DB>
output "mysql_connection" {
  description = "The connection string dynamically generated for storage inside the Kubernetes configmap"
  value       = format("%s:%s:%s", var.project_id, var.region, google_sql_database_instance.trillian.name)
}

// Postgres DB username.
output "mysql_user" {
  description = "The Cloud SQL Instance User name"
  value       = google_sql_user.trillian.name
}

// Postgres DB password.
output "mysql_pass" {
  sensitive   = true
  description = "The Cloud SQL Instance Password (Generated)"
  value       = google_sql_user.trillian.password
}
