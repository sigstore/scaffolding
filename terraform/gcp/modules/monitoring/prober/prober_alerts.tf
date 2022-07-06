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

resource "google_monitoring_alert_policy" "prober_rekor_endpoint_latency" {
  alert_strategy {
    auto_close = "604800s"
  }

  combiner = "OR"

  conditions {
    condition_threshold {
      aggregations {
        alignment_period     = "300s"
        cross_series_reducer = "REDUCE_PERCENTILE_95"
        group_by_fields      = ["metric.label.endpoint"]
        per_series_aligner   = "ALIGN_MEAN"
      }

      comparison      = "COMPARISON_GT"
      duration        = "300s"
      filter          = format("resource.type = \"k8s_container\" AND metric.type = \"external.googleapis.com/prometheus/api_endpoint_latency\" AND metric.labels.host = \"%s\"", var.rekor_url)
      threshold_value = "750"

      trigger {
        count   = "1"
        percent = "0"
      }
    }

    display_name = "API Prober: Rekor API Endpoint Latency > 750 ms"
  }

  display_name = "API Prober: Rekor API Endpoint Latency > 750 ms for 5 minutes"

  documentation {
    content   = "At least one supported Rekor API Endpoint has had latency > 750 ms for 5 minutes."
    mime_type = "text/markdown"
  }

  enabled               = "false"
  notification_channels = local.notification_channels
  project               = var.project_id
}

resource "google_monitoring_alert_policy" "prober_fulcio_endpoint_latency" {
  alert_strategy {
    auto_close = "604800s"
  }

  combiner = "OR"

  conditions {
    condition_threshold {
      aggregations {
        alignment_period     = "300s"
        cross_series_reducer = "REDUCE_PERCENTILE_95"
        group_by_fields      = ["metric.label.endpoint"]
        per_series_aligner   = "ALIGN_MEAN"
      }

      comparison      = "COMPARISON_GT"
      duration        = "300s"
      filter          = format("resource.type = \"k8s_container\" AND metric.type = \"external.googleapis.com/prometheus/api_endpoint_latency\" AND metric.labels.host = \"%s\"", var.fulcio_url)
      threshold_value = "750"

      trigger {
        count   = "1"
        percent = "0"
      }
    }

    display_name = "API Prober: Fulcio API Endpoint Latency > 750 ms"
  }

  display_name = "API Prober: Fulcio API Endpoint Latency > 750 ms for 5 minutes"

  documentation {
    content   = "At least one supported Fulcio API Endpoint has had latency > 750 ms for 5 minutes."
    mime_type = "text/markdown"
  }

  enabled               = "true"
  notification_channels = local.notification_channels
  project               = var.project_id
}

resource "google_monitoring_alert_policy" "prober_data_absent_alert" {
  for_each = toset(local.hosts)

  alert_strategy {
    auto_close = "604800s"
  }

  combiner = "OR"

  conditions {
    condition_absent {
      aggregations {
        alignment_period     = "300s"
        cross_series_reducer = "REDUCE_PERCENTILE_95"
        group_by_fields      = ["metric.label.endpoint"]
        per_series_aligner   = "ALIGN_MEAN"
      }

      duration = "300s"
      filter   = format("resource.type = \"k8s_container\" AND metric.type = \"external.googleapis.com/prometheus/api_endpoint_latency\" AND metric.labels.host = \"%s\"", each.key)

      trigger {
        count   = "1"
        percent = "0"
      }
    }

    display_name = format("API Prober: Latency Data Absent for 5 minutes: %s", each.key)
  }

  display_name = format("API Prober: Latency Data Absent for 5 minutes: %s", each.key)

  documentation {
    content   = format("API Endpoint Latency Data Absent for 5 minutes: %s. Check playbook for more details.", each.key)
    mime_type = "text/markdown"
  }

  enabled               = "true"
  notification_channels = local.notification_channels
  project               = var.project_id
}

resource "google_monitoring_alert_policy" "prober_error_codes" {
  alert_strategy {
    auto_close = "604800s"
  }

  combiner = "OR"

  conditions {
    condition_threshold {
      aggregations {
        alignment_period     = "300s"
        cross_series_reducer = "REDUCE_MAX"
        group_by_fields      = ["metric.label.endpoint"]
        per_series_aligner   = "ALIGN_RATE"
      }

      comparison      = "COMPARISON_GT"
      duration        = "0s"
      filter          = "resource.type = \"k8s_container\" AND metric.type = \"external.googleapis.com/prometheus/api_endpoint_latency_count\" AND metric.labels.status_code != \"200\""
      threshold_value = "0"

      trigger {
        count   = "1"
        percent = "0"
      }
    }

    display_name = "API Prober: Error Codes are non-200"
  }

  display_name = "API Prober: Error Codes are non-200"

  documentation {
    content   = "At least one Sigstore API endpoint has returned non-200 error codes for at least 5 minutes.\n"
    mime_type = "text/markdown"
  }

  enabled               = "true"
  notification_channels = local.notification_channels
  project               = var.project_id
}
