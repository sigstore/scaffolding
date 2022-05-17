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

// URLs for Sigstore services
variable "rekor_url" {
  description = "Rekor URL"
  default     = "rekor.sigstore.dev"
}

// Set-up for notification channel for alerting
variable "notification_channel_id" {
  type        = string
  description = "The notification channel ID which alerts should be sent to. You can find this by running `gcloud alpha monitoring channels list`."
}

variable "api_endpoints_get" {
  type = list(string)
  default = [
    "/",
    "/api/v1/version",
    "/api/v1/log",
    "/api/v1/log/publicKey",
  ]
}

locals {
  notification_channels = [format("projects/%v/notificationChannels/%v", var.project_id, var.notification_channel_id)]
}

