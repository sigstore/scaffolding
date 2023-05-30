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

// Alert if we see a failure every minute for 5 consecutive minutes
resource "google_monitoring_alert_policy" "fulcio_uptime_alert" {
  # In the absence of data, incident will auto-close in 7 days
  alert_strategy {
    auto_close = "604800s"
  }
  combiner = "OR"

  conditions {
    condition_threshold {
      aggregations {
        alignment_period     = "60s"
        cross_series_reducer = "REDUCE_COUNT_FALSE"
        group_by_fields      = ["resource.*"]
        per_series_aligner   = "ALIGN_NEXT_OLDER"
      }

      comparison      = "COMPARISON_GT"
      duration        = "300s"
      filter          = format("metric.type=\"monitoring.googleapis.com/uptime_check/check_passed\" resource.type=\"uptime_url\" metric.label.\"check_id\"=\"%s\"", google_monitoring_uptime_check_config.uptime_fulcio.uptime_check_id)
      threshold_value = "1"

      trigger {
        count   = "1"
        percent = "0"
      }
    }

    display_name = "Failure of uptime check_id fulcio-uptime"
  }

  display_name          = "Fulcio uptime alert"
  enabled               = "true"
  notification_channels = local.notification_channels
  project               = var.project_id
  depends_on            = [google_monitoring_uptime_check_config.uptime_fulcio]
}

// Alert if we see a failure every minute for 5 consecutive minutes
resource "google_monitoring_alert_policy" "ctlog_uptime_alert" {
  # In the absence of data, incident will auto-close in 7 days
  alert_strategy {
    auto_close = "604800s"
  }
  combiner = "OR"

  conditions {
    condition_threshold {
      aggregations {
        alignment_period     = "60s"
        cross_series_reducer = "REDUCE_COUNT_FALSE"
        group_by_fields      = ["resource.*"]
        per_series_aligner   = "ALIGN_NEXT_OLDER"
      }

      comparison      = "COMPARISON_GT"
      duration        = "300s"
      filter          = format("metric.type=\"monitoring.googleapis.com/uptime_check/check_passed\" resource.type=\"uptime_url\" metric.label.\"check_id\"=\"%s\"", google_monitoring_uptime_check_config.uptime_ct_log.uptime_check_id)
      threshold_value = "1"

      trigger {
        count   = "1"
        percent = "0"
      }
    }

    display_name = "Failure of uptime check_id ctlog-uptime"
  }

  display_name          = "CT Log Uptime Alert"
  enabled               = "true"
  notification_channels = local.notification_channels
  project               = var.project_id
  depends_on            = [google_monitoring_uptime_check_config.uptime_fulcio]
}

### Certificate Authority Alerts

# Certificate Authority Cert Expiration -- alert when cert will expire within 10 weeks
resource "google_monitoring_alert_policy" "ca_service_cert_expiration_alert" {
  # In the absence of data, incident will auto-close in 7 days
  alert_strategy {
    auto_close = "604800s"
  }

  combiner = "OR"

  conditions {
    condition_threshold {
      aggregations {
        alignment_period   = "300s"
        per_series_aligner = "ALIGN_MEAN"
      }

      comparison = "COMPARISON_LT"
      duration   = "0s"
      filter     = "metric.type=\"privateca.googleapis.com/ca/cert_expiration\" resource.type=\"privateca.googleapis.com/CertificateAuthority\""
      // alert on 10 weeks
      threshold_value = "6048000"

      trigger {
        count   = "1"
        percent = "0"
      }
    }

    display_name = "CA certificate expiration [MEAN]"
  }

  display_name = "Certificate Authority Cert Expiration"

  documentation {
    content   = "Certificate authority certs will expire in 10 weeks. Please rotate the root cert."
    mime_type = "text/markdown"
  }

  user_labels = {
    severity = "warning"
  }

  enabled               = "true"
  notification_channels = local.notification_channels
  project               = var.project_id
}

resource "google_monitoring_alert_policy" "ca_service_cert_quota" {
  alert_strategy {
    auto_close = "604800s"
  }

  combiner = "OR"

  conditions {
    condition_threshold {
      aggregations {
        alignment_period   = "300s"
        per_series_aligner = "ALIGN_RATE"
      }

      comparison      = "COMPARISON_GT"
      duration        = "300s"
      filter          = format("metric.type=\"privateca.googleapis.com/ca/cert/create_count\" resource.type=\"privateca.googleapis.com/CertificateAuthority\" resource.label.\"ca_pool_id\"=\"%s\"", var.ca_pool_name)
      threshold_value = "25"

      trigger {
        count   = "1"
        percent = "0"
      }
    }

    display_name = "Certificate creation count for sigstore [RATE]"
  }

  display_name = "Certificate creation count for sigstore CA above quota"

  documentation {
    content   = "According to docs for the CA service, the DevOps tier CA has a [request quota](https://cloud.google.com/certificate-authority-service/quotas#request_quotas) of 25 certs/second.\n;\nThis alert will fire if we exceed 25 certs/second for longer than 5 minutes.\n\nIf this happens, consider increasing quotas as described [here](https://cloud.google.com/docs/quota#requesting_higher_quota)"
    mime_type = "text/markdown"
  }

  enabled               = "true"
  notification_channels = local.notification_channels
  project               = var.project_id
}

### K8s Alerts

resource "google_monitoring_alert_policy" "fulcio_k8s_pod_restart_failing_container" {
  # adding a dependency on the associated metric means that Terraform will 
  # always try to apply changes to the metric before this alert
  depends_on = [google_logging_metric.fulcio_k8s_pod_restart_failing_container]

  # In the absence of data, incident will auto-close in 7 days
  alert_strategy {
    auto_close = "604800s"
  }

  combiner = "OR"

  conditions {
    condition_threshold {
      aggregations {
        alignment_period   = "600s"
        per_series_aligner = "ALIGN_COUNT"
      }

      comparison              = "COMPARISON_GT"
      duration                = "600s"
      evaluation_missing_data = "EVALUATION_MISSING_DATA_NO_OP"
      filter                  = "metric.type=\"logging.googleapis.com/user/fulcio/k8s_pod/restarting-failed-container\" resource.type=\"k8s_pod\""
      threshold_value         = "1"

      trigger {
        count   = "1"
        percent = "0"
      }
    }

    display_name = "K8s Restart Failing Container for more than ten minutes"
  }

  display_name = "Fulcio K8s Restart Failing Container"

  documentation {
    content   = "K8s is restarting a failing container for longer than the accepted time limit, please see playbook for help.\n"
    mime_type = "text/markdown"
  }

  enabled               = "true"
  notification_channels = local.notification_channels
  project               = var.project_id
}

resource "google_monitoring_alert_policy" "fulcio_k8s_pod_unschedulable" {
  # adding a dependency on the associated metric means that Terraform will 
  # always try to apply changes to the metric before this alert
  depends_on = [google_logging_metric.k8s_pod_unschedulable]

  # In the absence of data, incident will auto-close in 7 days
  alert_strategy {
    auto_close = "604800s"
  }

  combiner = "OR"

  conditions {
    condition_threshold {
      aggregations {
        alignment_period     = "600s"
        cross_series_reducer = "REDUCE_COUNT"
        per_series_aligner   = "ALIGN_COUNT"
      }

      comparison              = "COMPARISON_GT"
      duration                = "600s"
      evaluation_missing_data = "EVALUATION_MISSING_DATA_NO_OP"
      filter                  = "metric.type=\"logging.googleapis.com/user/fulcio/k8s_pod/unschedulable\" resource.type=\"k8s_pod\""
      threshold_value         = "1"

      trigger {
        count   = "1"
        percent = "0"
      }
    }

    display_name = "K8s was unable to schedule a pod for more than ten minutes"
  }

  display_name = "Fulcio K8s Unscheduable"

  documentation {
    content   = "K8s is failing to schedulable pod for longer than the accepted time limit, please see playbook for help.\n"
    mime_type = "text/markdown"
  }

  enabled               = "true"
  notification_channels = local.notification_channels
  project               = var.project_id
}
