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

variable "region" {
  type        = string
  description = "GCP region"
  default     = "us-west1"
}

variable "replica_zones" {
  description = "List of zones for read replicas."
  type        = list(any)
  default     = []
}

variable "cluster_name" {
  type    = string
  default = ""
}

variable "tier" {
  type        = string
  description = "Machine tier for MySQL instance."
  default     = "db-n1-standard-1"
}

variable "replica_tier" {
  type        = string
  description = "Machine tier for MySQL replica."
  default     = "db-n1-standard-1"
}

variable "availability_type" {
  type        = string
  description = "Availability tier for MySQL"
  default     = "REGIONAL"
}

variable "ipv4_enabled" {
  type        = bool
  description = "Whether to enable ipv4 for MySQL instance."
  default     = false
}

variable "require_ssl" {
  type        = bool
  description = "Whether to require ssl for MySQL instance."
  default     = true
}

variable "backup_enabled" {
  type        = bool
  description = "Whether to enable backup configuration."
  default     = true
}

variable "binary_log_backup_enabled" {
  type        = bool
  description = "Whether to enable binary log for backup."
  default     = true
}

variable "network" {
  type    = string
  default = "default"
}

variable "subnetwork" {
  type    = string
  default = "default"
}

variable "instance_name" {
  type        = string
  description = "Name for MySQL instance."
}

variable "db_name" {
  type        = string
  description = "Name for MySQL database name."
  default     = "trillian"
}

variable "database_version" {
  type        = string
  description = "MySQL database version."
  default     = "MYSQL_8_0_37"
}

variable "deletion_protection" {
  type        = bool
  description = "Deletion protection for MYSQL database. Must be set to false for `terraform apply` or `terraform destroy` to delete the db."
  default     = true
}

// This is ok to have here because nobody can connect to the database without
// IAM and it sits on private network.
variable "password" {
  type        = string
  description = "mysql password within the database"
  sensitive   = true
}

variable "collation" {
  type        = string
  description = "collation setting for database"
  default     = "utf8mb3_general_ci"
}
