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

data "google_project" "project" {
  project_id = var.project_id
}

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
  default_page_configuration = {
    fast_burn   = true
    medium_burn = true
    slow_burn   = false
  }

  flattened_availability_slos = flatten([
    for category_id, availability_slo_category in var.availability_slos : [
      for slo_id, slo in availability_slo_category.slos : {
        slo_id               = "${category_id}-${slo_id}"
        display_name         = "${format("%.1f%%", slo.goal * 100)} ${availability_slo_category.display_prefix} : ${slo.display_suffix}"
        goal                 = slo.goal
        bad_service_filter   = "${availability_slo_category.base_total_service_filter} ${availability_slo_category.bad_filter} ${slo.label_filter}"
        total_service_filter = "${availability_slo_category.base_total_service_filter} ${slo.label_filter}"
        page_configuration   = coalesce(slo.page_configuration, coalesce(availability_slo_category.page_configuration, local.default_page_configuration))
      }
    ]
  ])

  # Implemented recommended burn rates:
  # https://sre.google/workbook/alerting-on-slos/#recommended_time_windows_and_burn_rates_f
  slow_burn_alerts = [
    for slo in local.flattened_availability_slos : {
      alert_id            = "${slo.slo_id}-slow"
      display_name        = "SLO Slow Burn: ${var.display_name} ${slo.display_name}"
      window              = "72h"
      burn_rate_threshold = 1
      slo_id              = slo.slo_id
      page                = slo.page_configuration.slow_burn
    }
  ]

  medium_burn_alerts = [
    for slo in local.flattened_availability_slos : {
      alert_id            = "${slo.slo_id}-medium"
      display_name        = "SLO Medium Burn: ${var.display_name} ${slo.display_name}"
      window              = "6h"
      burn_rate_threshold = 6
      slo_id              = slo.slo_id
      page                = slo.page_configuration.medium_burn
    }
  ]

  fast_burn_alerts = [
    for slo in local.flattened_availability_slos : {
      alert_id            = "${slo.slo_id}-fast"
      display_name        = "SLO Fast Burn: ${var.display_name} ${slo.display_name}"
      window              = "1h"
      burn_rate_threshold = 14.4
      slo_id              = slo.slo_id
      page                = slo.page_configuration.fast_burn
    }
  ]
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

resource "google_monitoring_alert_policy" "availability_burn_alert" {
  for_each = {
    for alert in concat(local.slow_burn_alerts, local.medium_burn_alerts, local.fast_burn_alerts) :
    "${alert.alert_id}" => alert
  }
  project = var.project_id

  display_name = each.value.display_name
  combiner     = "AND"
  conditions {
    display_name = each.value.display_name
    condition_threshold {
      filter = format("select_slo_burn_rate(\"projects/%s/services/%s/serviceLevelObjectives/%s\", %s)", data.google_project.project.number, google_monitoring_custom_service.service.service_id,
      each.value.slo_id, each.value.window)
      threshold_value = each.value.burn_rate_threshold
      duration        = "0s"
      comparison      = "COMPARISON_GT"
    }
  }

  notification_channels = each.value.page ? var.notification_channels : []

  documentation {
    content = format("SLO burn rate for the past %s exceeded %s times the acceptable error budget rate.", each.value.window, each.value.burn_rate_threshold)
  }
}
