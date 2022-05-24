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
  type = string
  validation {
    condition     = length(var.project_id) > 0
    error_message = "Must specify project_id variable."
  }
}

variable "region" {
  description = "The region in which to create the VPC network"
  type        = string
}

variable "tuf_region" {
  description = "The region in which to create the TUF bucket"
  type        = string
  default     = ""
}

variable "attestation_region" {
  description = "The region in which to create the attestation bucket"
  type        = string
  default     = ""
}

variable "attestation_bucket" {
  type        = string
  description = "Name of GCS bucket for attestation."
}

variable "attestation_storage_class" {
  type        = string
  description = "Storage class for attestation bucket."
  default     = "REGIONAL"
}

variable "tuf_bucket" {
  type        = string
  description = "Name of GCS bucket for TUF root."
}

variable "tuf_storage_class" {
  type        = string
  description = "Storage class for TUF bucket."
  default     = "REGIONAL"
}

variable "ca_pool_name" {
  description = "Certificate authority pool name"
  type        = string
  default     = "sigstore"
}

variable "ca_name" {
  description = "Certificate authority name"
  type        = string
  default     = "sigstore-authority"
}

variable "monitoring" {
  description = "Monitoring and alerting"
  type = object({
    enabled                 = bool
    fulcio_url              = string
    rekor_url               = string
    dex_url                 = string
    notification_channel_id = string
  })
  default = {
    enabled                 = false
    fulcio_url              = "fulcio.example.com"
    rekor_url               = "rekor.example.com"
    dex_url                 = "oauth2.example.com"
    notification_channel_id = ""
  }
}

// Optional values that can be overridden or appended to if desired.
variable "cluster_name" {
  description = "The name to give the new Kubernetes cluster."
  type        = string
  default     = "sigstore-staging"
}

variable "cluster_network_tag" {
  type    = string
  default = ""
}

variable "tunnel_accessor_sa" {
  type        = string
  description = "Email of group to give access to the tunnel to"
}

variable "github_repo" {
  description = "Github repo for running Github Actions from."
  type        = string
}

variable "mysql_instance_name" {
  type        = string
  description = "Name for MySQL instance. If unspecified, will default to '[var.cluster-name]-mysql-[random.suffix]'"
  default     = ""
}

variable "mysql_db_name" {
  type        = string
  description = "Name for MySQL database name."
  default     = "trillian"
}

variable "mysql_db_version" {
  type        = string
  description = "MySQL database version."
  default     = "MYSQL_5_7"
}

variable "mysql_tier" {
  type        = string
  description = "Machine tier for MySQL instance."
  default     = "db-n1-standard-1"
}

variable "mysql_availability_type" {
  type        = string
  description = "Availability tier for MySQL"
  default     = "REGIONAL"
}

variable "mysql_ipv4_enabled" {
  type        = bool
  description = "Whether to enable ipv4 for MySQL instance."
  default     = false
}

variable "mysql_require_ssl" {
  type        = bool
  description = "Whether to require ssl for MySQL instance."
  default     = true
}

variable "mysql_backup_enabled" {
  type        = bool
  description = "Whether to enable backup configuration for MySQL instance."
  default     = true
}

variable "mysql_binary_log_backup_enabled" {
  type        = bool
  description = "Whether to enable binary log for backup for MySQL instance."
  default     = true
}

variable "fulcio_keyring_name" {
  type        = string
  description = "Name of Fulcio keyring."
  default     = "fulcio-keyring"
}

variable "fulcio_intermediate_key_name" {
  type        = string
  description = "Name of Fulcio intermediate key."
  default     = "fulcio-intermediate-key"
}

variable "rekor_keyring_name" {
  type        = string
  description = "Name of Rekor keyring."
  default     = "rekor-keyring"
}

variable "rekor_key_name" {
  type        = string
  description = "Name of Rekor key."
  default     = "rekor-key"
}
