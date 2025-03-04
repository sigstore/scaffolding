/**
 * Copyright 2025 The Sigstore Authors
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

variable "project_number" {
  type    = string
  default = ""
  validation {
    condition     = length(var.project_number) > 0
    error_message = "Must specify PROJECT_NUMBER variable."
  }
}

variable "cluster_location" {
  description = "Zone or Region to create cluster in."
  type        = string
  default     = "us-central1"
}

// Optional values that can be overridden or appended to if desired.
variable "cluster_name" {
  description = "The name of the Kubernetes cluster."
  type        = string
  default     = ""
}

// URLs for Sigstore services
variable "tsa_url" {
  description = "TSA URL"
  default     = "tsa.sigstore.dev"
}

variable "prober_url" {
  description = "TSA Prober URL"
  type        = string
  default     = ""
}

// Namespace for monitored service
variable "gke_namespace" {
  description = "GKE Namespace"
  type        = string
  default     = "tsa-system"
}

// Set-up for notification channel for alerting
variable "notification_channel_ids" {
  type        = list(string)
  description = "List of notification channel IDs which alerts should be sent to. You can find this by running `gcloud alpha monitoring channels list`."
}

locals {
  notification_channels = toset([for nc in var.notification_channel_ids : format("projects/%v/notificationChannels/%v", var.project_id, nc)])
}

variable "create_slos" {
  description = "True to enable SLO creation"
  type        = bool
  default     = false
}
