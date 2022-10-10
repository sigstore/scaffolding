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
  description = "Project ID in which the monitored service lives."
  type        = string
  default     = ""
  validation {
    condition     = length(var.project_id) > 0
    error_message = "Must specify PROJECT_ID variable."
  }
}

variable "service_id" {
  description = "Resource ID for the monitoring service."
  type        = string
  default     = ""
  validation {
    condition     = length(var.service_id) > 0
    error_message = "Must specify serivce_id variable."
  }
}

variable "display_name" {
  description = "Friendly name for the monitored service."
  type        = string
  default     = ""
}

variable "resource_name" {
  description = "Resource name (e.g. k8s namespace) for telemetry and logs of the service."
  type        = string
  default     = ""
}

variable "notification_channels" {
  description = "Set of preformated notification channel resource ids"
  type        = set(string)
  default     = []
}

/* availability_slos define 30-day window, request based availability SLOs, grouped by category.
 *
 * Example usage:
 *
 * availability_slos = {
 *   server-slo {
 *     display_prefix            = "Server Availability"
 *     base_total_service_filter = "metric.type=\"<metric_uri>\" resource.type=\"prometheus_target\""
 *     bad_filter                = "metrics.labels.status!=\"OK\""
 *     slos {
 *       rpc1 {
 *         goal           = 0.995
 *         display_suffix = "RPC1"
 *         label_filter   = "metrics.labels.rpc = \"RPC1\""
 *
 *       },
 *       rpc2 {
 *         ...
 *       },
 *       ...
 *     },
 *   },
 *   prober-slo = {
 *     ...
 *     slos {
 *       rpc1 {
 *         ...
 *       }
 *     ...
 *   },
 * }
 *
 *
 * Defines SLOs with:
 *
 * 1.
 * slo_id               = "server-slo-rpc1"
 * display_name         = "99.5% Server Availability RPC1"
 * total_service_filter = "metric.type=\"<metric_uri>\" resource.type=\"prometheus_target\" metrics.labels.rpc = \"RPC1\""
 * bad_service_filter   = "metric.type=\"<metric_uri>\" resource.type=\"prometheus_target\" metrics.labels.rpc = \"RPC1\" metrics.labels.status!=\"OK\""
 * goal                 = 0.995
 *
 * 2.
 * slo_id               = "server-slo-rpc2"
 * ...
 *
 * n.
 * slo_id               = "prober-slo-rpc1"
 * ...
 *
 * Note: map keys are concatenated together with a hyphen to create the SLO id. SLO
 * ids must match "^[a-z0-9\\-]+$" or they will be rejected at create time.
 */
variable "availability_slos" {
  type = map(object(
    {
      display_prefix            = string
      base_total_service_filter = string
      bad_filter                = string
      page_configuration = optional(object({
        fast_burn   = bool
        medium_burn = bool
        slow_burn   = bool
      }))
      slos = map(object({
        goal           = number
        display_suffix = string
        label_filter   = string
        page_configuration = optional(object({
          fast_burn   = bool
          medium_burn = bool
          slow_burn   = bool
        }))
      }))
  }))
  default = {}
}
