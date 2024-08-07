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

// Outputs a list of strings for each CTLog Cloud SQL instance.
output "ctlog_mysql_instances" {
  description = "Names of the DB instances created for the CTLog shards"
  value       = [for ctlog_shard in module.ctlog_shards : ctlog_shard.mysql_instance]
}

// Outputs a list of connection strings for each CTLog Cloud SQL instance.
output "ctlog_mysql_connections" {
  description = "Connection strings of the DB instances created for the CTLog shards"
  value       = [for ctlog_shard in module.ctlog_shards : ctlog_shard.mysql_connection]
}

// Outputs a list of strings for each Standalone Cloud SQL instance.
output "standalone_mysql_instances" {
  description = "Names of the DB instances created for the standalone MySQLs"
  value       = [for standalone in module.standalone_mysqls : standalone.mysql_instance]
}

// Outputs a list of connection strings for each Standalone Cloud SQL instance.
output "standalone_mysql_connections" {
  description = "Connection strings of the DB instances created for the standalone MySQLs"
  value       = [for standalone in module.standalone_mysqls : standalone.mysql_connection]
}

// Full connection string for the MySQL DB>
output "mysql_connection" {
  description = "The connection string dynamically generated for storage inside the Kubernetes configmap"
  value       = module.mysql.mysql_connection
}

// MySQL DB username.
output "mysql_user" {
  description = "The Cloud SQL Instance User name"
  value       = module.mysql.mysql_user
}

// MySQL DB password.
output "mysql_pass" {
  sensitive   = true
  description = "The Cloud SQL Instance Password (Generated)"
  value       = module.mysql.mysql_pass
}

// CTLog MySQL DB name.
output "ctlog_mysql_database" {
  description = "The CTLog Cloud SQL Database name"
  value       = length(var.ctlog_shards) == 0 ? null : element([for ctlog_shard in module.ctlog_shards : ctlog_shard.mysql_database], 0)
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

