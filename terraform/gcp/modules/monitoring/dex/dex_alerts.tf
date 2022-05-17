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

resource "google_monitoring_uptime_check_config" "uptime_dex" {
  display_name = "Dex Uptime"

  http_check {
    mask_headers   = "false"
    path           = "/auth/healthz"
    port           = "80"
    request_method = "GET"
    use_ssl        = "false"
    validate_ssl   = "false"
  }

  monitored_resource {
    labels = {
      host       = var.dex_url
      project_id = var.project_id
    }

    type = "uptime_url"
  }

  period  = "60s"
  project = var.project_id
  timeout = "10s"
}

# Alert for Dex uptime
resource "google_monitoring_alert_policy" "dex_uptime_alert" {
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

      comparison      = "COMPARISON_GT"
      duration        = "60s"
      filter          = format("metric.type=\"monitoring.googleapis.com/uptime_check/check_passed\" resource.type=\"uptime_url\" metric.label.\"check_id\"=\"%s\"", google_monitoring_uptime_check_config.uptime_dex.uptime_check_id)
      threshold_value = "1"

      trigger {
        count   = "1"
        percent = "0"
      }
    }

    display_name = "Failure of uptime check_id dex-uptime"
  }

  display_name          = "Dex uptime alert"
  enabled               = "true"
  notification_channels = local.notification_channels
  project               = var.project_id
  depends_on            = [google_monitoring_uptime_check_config.uptime_dex]
}
