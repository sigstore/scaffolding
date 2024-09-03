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
}

// Storage variables
variable "tuf_bucket" {
  type        = string
  description = "Name of GCS bucket for TUF root."
}

variable "tuf_bucket_member" {
  type        = string
  description = "User, group, or service account to grant access to the TUF GCS buckets. Use 'allUsers' for general access, or e.g. group:mygroup@myorg.com for granular access."
  default     = "allUsers"
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

// Service account variables
variable "tuf_service_account_name" {
  type        = string
  description = "Name of service account for TUF signing on GitHub Actions"
  default     = "tuf-gha"
}

// KMS variables
variable "tuf_keyring_name" {
  type        = string
  description = "Name of KMS keyring for TUF metadata signing"
  default     = "tuf-keyring"
}

variable "tuf_key_name" {
  type        = string
  description = "Name of KMS key for TUF metadata signing"
  default     = "tuf-key"
}

variable "kms_location" {
  type        = string
  description = "Location of KMS keyring"
  default     = "global"
}

variable "tuf_key_viewers" {
  type        = list(string)
  description = "List of members who can view the public key. See https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/google_kms_key_ring_iam#argument-reference for supported values"
  default     = []
}

variable "main_page_suffix" {
  type        = string
  description = "Behaves as the bucket's directory index where missing objects are treated as potential directories"
  default     = ""
}
