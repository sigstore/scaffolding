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

variable "project_id" {
  type    = string
  default = ""
  validation {
    condition     = length(var.project_id) > 0
    error_message = "Must specify project_id variable."
  }
}

variable "dns_zone_name" {
  description = "Name of DNS Zone object in Google Cloud DNS"
  type        = string
}

variable "dns_domain_name" {
  description = "Name of DNS domain name in Google Cloud DNS"
  type        = string
}

variable "load_balancer_ipv4" {
  description = "IPv4 adddress of external load balancer"
  type        = string
}

variable "region" {
  description = "The region in which to create the VPC network"
  type        = string
  default     = "us-west1"
}

variable "enable_ctlog_sql" {
  description = "Enable a database module for creating/editing Cloud SQL Instance"
  type        = bool
  default     = false
}

// Optional values that can be overridden or appended to if desired.
variable "cluster_name" {
  description = "The name to give the new Kubernetes cluster."
  type        = string
  default     = "sigstore-staging"
}

variable "network" {
  type        = string
  description = "Network to connect to."
  default     = "default"
}

variable "mysql_instance_name" {
  type        = string
  description = "Name for CTLog MySQL instance. If unspecified, will default to '[var.cluster-name]-ctlog-mysql-[random.suffix]'"
  default     = ""
}

variable "mysql_db_name" {
  type        = string
  description = "Name for CTLog MySQL database name."
  default     = "trillian"
}

variable "mysql_db_version" {
  type        = string
  description = "CTLog MySQL database version."
  default     = "MYSQL_5_7"
}

variable "mysql_tier" {
  type        = string
  description = "Machine tier for CTLog MySQL instance."
  default     = "db-n1-standard-1"
}

variable "mysql_availability_type" {
  type        = string
  description = "Availability tier for CTLog MySQL"
  default     = "REGIONAL"
}

variable "mysql_replica_zones" {
  description = "List of zones for read replicas."
  type        = list(any)
  default     = []
}

variable "mysql_replica_tier" {
  type        = string
  description = "Machine tier for CTLog MySQL replica."
  default     = "db-n1-standard-1"
}

variable "mysql_ipv4_enabled" {
  type        = bool
  description = "Whether to enable ipv4 for CTLog MySQL instance."
  default     = false
}

variable "mysql_require_ssl" {
  type        = bool
  description = "Whether to require ssl for CTLog MySQL instance."
  default     = true
}

variable "mysql_backup_enabled" {
  type        = bool
  description = "Whether to enable backup configuration for CTLog MySQL instance."
  default     = true
}

variable "mysql_binary_log_backup_enabled" {
  type        = bool
  description = "Whether to enable binary log for backup for CTLog MySQL instance."
  default     = true
}

variable "mysql_database_version" {
  type        = string
  description = "CTLog MySQL database version."
  default     = "MYSQL_5_7"
}
