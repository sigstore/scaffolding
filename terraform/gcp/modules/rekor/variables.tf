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
  description = "GCP region"
  type        = string
}

variable "cluster_name" {
  type    = string
  default = ""
}

variable "network" {
  type    = string
  default = "default"
}

// Storage
variable "enable_attestations" {
  type        = bool
  default     = true
  description = "enable/disable storage for attestations"
}

variable "attestation_bucket" {
  type        = string
  default     = ""
  description = "Name of GCS bucket for attestation."
}

variable "attestation_region" {
  type        = string
  description = "Attestation bucket region"
  default     = ""
}

variable "storage_class" {
  type        = string
  description = "Storage class for TUF root bucket."
  default     = "REGIONAL"
}

variable "gcs_logging_enabled" {
  type        = bool
  description = "enable/disable logging of GCS bucket traffic"
  default     = false
}

variable "gcs_logging_bucket" {
  description = "name of GCS bucket where storage logs will be written"
  type        = string
  default     = ""
}

// KMS
variable "rekor_keyring_name" {
  type        = string
  description = "Name of KMS keyring for Rekor"
  default     = "rekor-keyring"
}

variable "rekor_key_name" {
  type        = string
  description = "Name of KMS key for Rekor"
  default     = "rekor-key"
}

variable "kms_location" {
  type        = string
  description = "Location of KMS keyring"
  default     = "global"
}

variable "dns_zone_name" {
  description = "Name of DNS Zone object in Google Cloud DNS"
  type        = string
}

variable "dns_domain_name" {
  description = "Name of DNS domain name in Google Cloud DNS"
  type        = string
}

variable "redis_cluster_memory_size_gb" {
  description = "size of redis cluster expressed in whole GB"
  type        = number
  default     = 30
}

variable "new_entry_pubsub_consumers" {
  // If this list is empty, the PubSub resources will not be created.
  description = "The list of IAM principals that can subscribe to events about new entries in the log"
  type        = list(string)
  default     = []
}
