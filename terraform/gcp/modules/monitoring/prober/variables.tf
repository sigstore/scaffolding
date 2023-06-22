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
  default     = "https://rekor.sigstore.dev"
}

variable "fulcio_url" {
  description = "Fulcio URL"
  default     = "https://fulcio.sigstore.dev"
}

// Set-up for notification channel for alerting
variable "notification_channel_ids" {
  type        = list(string)
  description = "List of notification channel IDs which alerts should be sent to. You can find this by running `gcloud alpha monitoring channels list`."
}

variable "rekor_probed_endpoints" {
  description = "Allow list of probed endpoints to monitor/alert."
  type        = list(string)
  default = [
    "/api/v1/log",
    "/api/v1/log/entries",
    "/api/v1/log/entries/{entryUUID}",
    "/api/v1/log/entries/retrieve",
    "/api/v1/log/proof",
    "/api/v1/log/publicKey",
  ]
}

variable "fulcio_probed_endpoints" {
  description = "Allow list of probed endpoints to monitor/alert."
  type        = list(string)
  default = [
    "/api/v1/rootCert",
    "/api/v1/signingCert",
    "/api/v2/configuration",
    "/api/v2/trustBundle",
    # "/api/v2/signingCert", # TODO: probe the v2 cert endpoint
  ]
}

locals {
  notification_channels  = toset([for nc in var.notification_channel_ids : format("projects/%v/notificationChannels/%v", var.project_id, nc)])
  fulcio_endpoint_filter = format("metric.labels.endpoint = one_of(\"%s\")", join("\", \"", var.fulcio_probed_endpoints))
  rekor_endpoint_filter  = format("metric.labels.endpoint = one_of(\"%s\")", join("\", \"", var.rekor_probed_endpoints))
  all_endpoints_filter   = format("metric.labels.endpoint = one_of(\"%s\")", join("\", \"", distinct(concat(var.rekor_probed_endpoints, var.fulcio_probed_endpoints))))
  hosts = [{
    host            = var.fulcio_url
    endpoint_filter = local.fulcio_endpoint_filter
    }, {
    host            = var.rekor_url
    endpoint_filter = local.rekor_endpoint_filter
    }
  ]
}
