// Used to identify the cluster in validate.sh.
output "cluster_name" {
  description = "Convenience output to obtain the GKE Cluster name"
  value       = module.gke-cluster.cluster_name
}

output "trillian_serviceaccount" {
  description = "The email/name of the GCP service account"
  value       = module.mysql.trillian_serviceaccount
}

// Used when setting up the GKE cluster to talk to MySQL.
output "mysql_instance" {
  description = "The generated name of the Cloud SQL instance"
  value       = module.mysql.mysql_instance
}

// Full connection string for the MySQL DB>
output "mysql_connection" {
  description = "The connection string dynamically generated for storage inside the Kubernetes configmap"
  value       = module.mysql.mysql_connection
}

// Postgres DB username.
output "mysql_user" {
  description = "The Cloud SQL Instance User name"
  value       = module.mysql.mysql_user
}

// Postgres DB password.
output "mysql_pass" {
  sensitive   = true
  description = "The Cloud SQL Instance Password (Generated)"
  value       = module.mysql.mysql_pass
}

output "cluster_endpoint" {
  description = "Cluster endpoint"
  value       = module.gke-cluster.cluster_endpoint
}

output "cluster_ca_certificate" {
  sensitive   = true
  description = "Cluster ca certificate (base64 encoded)"
  value       = module.gke-cluster.cluster_ca_certificate
}

output "get_credentials" {
  description = "Gcloud get-credentials command"
  value       = format("gcloud container clusters get-credentials --project %s --region %s --internal-ip %s", var.project_id, var.region, module.gke-cluster.cluster_name)
}

output "bastion_socks_proxy_setup" {
  description = "Gcloud compute ssh to the bastion host command"
  value       = "${module.bastion.ssh_cmd} -- -N -D 8118"
}

output "bastion_ssh_cmd" {
  description = "Instructions to connect to bastion"
  value       = "gcloud compute ssh --zone ${module.bastion.zone} ${module.bastion.name} --tunnel-through-iap --project ${module.bastion.project}"
}

output "bastion_name" {
  description = "Name of the Bastion."
  value       = module.bastion.name
}

output "bastion_zone" {
  description = "GCP zone that the Bastion is in."
  value       = module.bastion.zone
}

output "bastion_kubectl" {
  description = "kubectl command using the local proxy once the bastion_ssh command is running"
  value       = "HTTPS_PROXY=socks5://localhost:8118 kubectl get pods --all-namespaces"
}
