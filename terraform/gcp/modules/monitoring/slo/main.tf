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

resource "google_monitoring_custom_service" "service" {
  project      = var.project_id
  service_id   = var.service_id
  display_name = var.display_name

  dynamic "telemetry" {
    for_each = length(var.resource_name) > 0 ? ["yes"] : []
    content {
      resource_name = var.resource_name
    }
  }
}

locals {
  flattened_availability_slos = flatten([
    for category_id, availability_slo_category in var.availability_slos : [
      for slo_id, slo in availability_slo_category.slos : {
        slo_id               = "${category_id}-${slo_id}"
        display_name         = "${format("%.1f%%", slo.goal * 100)} ${availability_slo_category.display_prefix} : ${slo.display_suffix}"
        goal                 = slo.goal
        bad_service_filter   = "${availability_slo_category.base_total_service_filter} ${availability_slo_category.bad_filter} ${slo.label_filter}"
        total_service_filter = "${availability_slo_category.base_total_service_filter} ${slo.label_filter}"
      }
    ]
  ])
}

resource "google_monitoring_slo" "availability_slo" {
  for_each = {
    for flattened_slo in local.flattened_availability_slos :
    "${flattened_slo.slo_id}" => flattened_slo
  }
  project = var.project_id
  service = google_monitoring_custom_service.service.service_id

  slo_id              = each.key
  goal                = each.value.goal
  rolling_period_days = 30
  display_name        = each.value.display_name

  request_based_sli {
    good_total_ratio {
      bad_service_filter   = each.value.bad_service_filter
      total_service_filter = each.value.total_service_filter
    }
  }
}
