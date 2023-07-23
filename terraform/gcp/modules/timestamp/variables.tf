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

variable "cluster_name" {
  description = "The name to give the new Kubernetes cluster."
  type        = string
}

// KMS
variable "timestamp_keyring_name" {
  type        = string
  description = "Name of KMS keyring for Timestamp Authority"
  default     = "timestamp-keyring"
}

variable "timestamp_encryption_key_name" {
  type        = string
  description = "Name of KMS key for encrypting Tink private key for Timestamp Authority"
  default     = "timestamp-encryption-key"
}

variable "timestamp_intermediate_ca_key_name" {
  type        = string
  description = "Name of KMS key for intermediate CA for Timestamp Authority"
  default     = "timestamp-intermediate-ca-key"
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
