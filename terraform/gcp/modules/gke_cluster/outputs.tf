// Used to identify the cluster in validate.sh.
output "cluster_name" {
  description = "Convenience output to obtain the GKE Cluster name"
  value       = google_container_cluster.cluster.name
}

output "cluster_endpoint" {
  description = "Cluster endpoint"
  value       = google_container_cluster.cluster.endpoint
}

output "cluster_ca_certificate" {
  sensitive   = true
  description = "Cluster ca certificate (base64 encoded)"
  value       = google_container_cluster.cluster.master_auth[0].cluster_ca_certificate
}

output "get_credentials" {
  description = "Gcloud get-credentials command"
  value       = format("gcloud container clusters get-credentials --project %s --region %s --internal-ip %s", var.project_id, var.region, google_container_cluster.cluster.name)
}

output "gke_sa_email" {
  value = google_service_account.gke-sa.email
}

output "ca_certificate" {
  value = format("gcloud container clusters get-credentials --project %s --region %s --internal-ip %s", var.project_id, var.region, google_container_cluster.cluster.name)
}
