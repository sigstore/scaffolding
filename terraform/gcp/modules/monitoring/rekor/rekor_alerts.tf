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

# Alert for Rekor uptime: GET
resource "google_monitoring_alert_policy" "rekor_uptime_alerts" {
  for_each = toset(var.api_endpoints_get)

  # In the absence of data, incident will auto-close in 7 days
  alert_strategy {
    auto_close = "604800s"
  }
  combiner = "OR"

  conditions {
    condition_threshold {
      aggregations {
        alignment_period     = "300s"
        cross_series_reducer = "REDUCE_COUNT_FALSE"
        group_by_fields      = ["resource.*"]
        per_series_aligner   = "ALIGN_NEXT_OLDER"
      }

      comparison = "COMPARISON_GT"
      duration   = "60s"
      filter     = format("metric.type=\"monitoring.googleapis.com/uptime_check/check_passed\" resource.type=\"uptime_url\" metric.label.\"check_id\"=\"%s\"", google_monitoring_uptime_check_config.rekor_uptime_alerts_get[format("%s", each.key)].uptime_check_id)

      threshold_value = "1"

      trigger {
        count   = "1"
        percent = "0"
      }
    }

    display_name = format("Failure of uptime check_id %s", format("rekor-uptime%s", replace(each.key, "/", "-")))
  }

  display_name          = format("Rekor uptime alert - %s", each.key)
  enabled               = "true"
  notification_channels = local.notification_channels
  project               = var.project_id
  depends_on            = [google_monitoring_uptime_check_config.rekor_uptime_alerts_get]
}


# Rekor API Latency > 750ms for 5 minutes in any region
resource "google_monitoring_alert_policy" "rekor_api_latency_alert" {
  for_each = toset(var.api_endpoints_get)
  # In the absence of data, incident will auto-close in 7 days
  alert_strategy {
    auto_close = "604800s"
  }

  combiner = "OR"

  conditions {
    condition_threshold {
      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MAX"
      }

      comparison = "COMPARISON_GT"
      duration   = "300s"
      filter     = format("metric.type=\"monitoring.googleapis.com/uptime_check/request_latency\" resource.type=\"uptime_url\" metric.label.\"check_id\"=\"%s\"", google_monitoring_uptime_check_config.rekor_uptime_alerts_get[format("%s", each.key)].uptime_check_id)

      threshold_value = "750"

      trigger {
        count   = "1"
        percent = "0"
      }
    }

    display_name = format("Rekor API Latency > 750ms for 5 minutes in any region - %s", each.key)
  }

  display_name = format("Rekor API Latency > 750ms for 5 minutes in any region - %s", each.key)

  documentation {
    content   = "This alert triggered because Rekor API Latency is greater than 750ms for 5 minutes in any of the available regions."
    mime_type = "text/markdown"
  }

  enabled               = "true"
  notification_channels = local.notification_channels
  project               = var.project_id
  depends_on            = [google_monitoring_alert_policy.rekor_uptime_alerts]
}

