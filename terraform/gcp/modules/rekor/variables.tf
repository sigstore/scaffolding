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
variable "attestation_bucket" {
  type        = string
  description = "Name of GCS bucket for attestation."
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
