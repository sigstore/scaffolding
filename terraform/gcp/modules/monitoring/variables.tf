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
    error_message = "Must specify PROJECT_ID variable."
  }
}

variable "cluster_location" {
  type        = string
  description = "Zone or Region to create cluster in."
  default     = "us-central1"
}

// Optional values that can be overridden or appended to if desired.
variable "cluster_name" {
  description = "The name of the Kubernetes cluster."
  type        = string
  default     = ""
}

// URLs for Sigstore services
variable "fulcio_url" {
  description = "Fulcio URL"
  default     = "fulcio.sigstore.dev"
}

variable "rekor_url" {
  description = "Rekor URL"
  default     = "rekor.sigstore.dev"
}

variable "ctlog_url" {
  description = "CT Log URL"
  default     = "ctfe.sigstore.dev"
}

variable "dex_url" {
  description = "Dex URL"
  default     = "oauth2.sigstore.dev"
}

// Prober variables
variable "prober_rekor_url" {
  type    = string
  default = "http://rekor-server.rekor-system.svc"
}

variable "prober_fulcio_url" {
  type    = string
  default = "http://fulcio-server.fulcio-system.svc"
}

// Set-up for notification channel for alerting
variable "notification_channel_ids" {
  type        = list(string)
  description = "List of notification channel IDs which alerts should be sent to. You can find this by running `gcloud alpha monitoring channels list`."
}

locals {
  notification_channels = toset([for nc in var.notification_channel_ids : format("projects/%v/notificationChannels/%v", var.project_id, nc)])
  qualified_rekor_url   = format("https://%s", var.rekor_url)
  qualified_fulcio_url  = format("https://%s", var.fulcio_url)
}

// Certificate Authority name for alerting
variable "ca_pool_name" {
  description = "Certificate authority pool name"
  type        = string
  default     = "sigstore"
}

variable "create_slos" {
  description = "Creates SLOs when true."
  type        = bool
  default     = false
}
