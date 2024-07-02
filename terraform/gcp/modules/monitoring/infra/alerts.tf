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

# This file contains alerts generic to the Sigstore project
# Alerts specific to fulcio, rekor or dex should be in the appropriate `modules/monitoring/[service]` directory

### SSL Alerts

# SSL certificate expiring soon for uptime checks
resource "google_monitoring_alert_policy" "ssl_cert_expiry_alert" {
  combiner = "OR"

  conditions {
    condition_threshold {
      aggregations {
        alignment_period     = "300s"
        cross_series_reducer = "REDUCE_MEAN"
        group_by_fields      = ["resource.label.*"]
        per_series_aligner   = "ALIGN_NEXT_OLDER"
      }

      comparison = "COMPARISON_LT"
      duration   = "600s"
      filter     = "metric.type=\"monitoring.googleapis.com/uptime_check/time_until_ssl_cert_expires\" AND resource.type=\"uptime_url\""
      // Alert 4 weeks in advance
      threshold_value = "28"

      trigger {
        count   = "1"
        percent = "0"
      }
    }

    display_name = "SSL certificate expiring in 4 weeks for uptime checks"
  }

  documentation {
    content   = "SSL certificates for at least one of the uptime checks will expire within 4 weeks. Please renew the certificate."
    mime_type = "text/markdown"
  }

  display_name          = "SSL certificate expiring in 4 weeks for uptime checks"
  enabled               = "true"
  notification_channels = local.notification_channels
  project               = var.project_id

  user_labels = {
    uptime   = "ssl_cert_expiration"
    version  = "1"
    severity = "warning"
  }
}

### Cloud SQL Alerts

# Cloud SQL Database CPU Utilization > 80%
resource "google_monitoring_alert_policy" "cloud_sql_cpu_utilization_warning" {
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
      comparison      = "COMPARISON_GT"
      duration        = "0s"
      filter          = "metric.type=\"cloudsql.googleapis.com/database/cpu/utilization\" resource.type=\"cloudsql_database\""
      threshold_value = "0.8"
      trigger {
        count   = "1"
        percent = "0"
      }
    }
    display_name = "Cloud SQL Database - CPU Utilization [MEAN]"
  }
  display_name = "Cloud SQL Database CPU Utilization > 80%"
  documentation {
    content   = "Cloud SQL Database CPU Utilization is >80%. Please increase CPU capacity via the database tier (https://cloud.google.com/sql/docs/mysql/instance-settings)."
    mime_type = "text/markdown"
  }
  enabled               = "true"
  notification_channels = local.notification_channels
  project               = var.project_id

  user_labels = {
    severity = "warning"
  }
}

# Cloud SQL Database CPU Utilization > 90%
resource "google_monitoring_alert_policy" "cloud_sql_cpu_utilization" {
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
      comparison      = "COMPARISON_GT"
      duration        = "0s"
      filter          = "metric.type=\"cloudsql.googleapis.com/database/cpu/utilization\" resource.type=\"cloudsql_database\""
      threshold_value = "0.9"
      trigger {
        count   = "1"
        percent = "0"
      }
    }
    display_name = "Cloud SQL Database - CPU Utilization [MEAN]"
  }
  display_name = "Cloud SQL Database CPU Utilization > 90%"
  documentation {
    content   = "Cloud SQL Database CPU Utilization is >90%. Please increase CPU capacity via the database tier (https://cloud.google.com/sql/docs/mysql/instance-settings)."
    mime_type = "text/markdown"
  }
  enabled               = "true"
  notification_channels = local.notification_channels
  project               = var.project_id
}

# Cloud SQL Database Memory Utilization > 90%
resource "google_monitoring_alert_policy" "cloud_sql_memory_utilization_warning" {
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

      comparison      = "COMPARISON_GT"
      duration        = "0s"
      filter          = "metric.type=\"cloudsql.googleapis.com/database/memory/utilization\" resource.type=\"cloudsql_database\""
      threshold_value = "0.9"

      trigger {
        count   = "1"
        percent = "0"
      }
    }

    display_name = "Cloud SQL Database - Memory utilization [MEAN]"
  }

  display_name = "Cloud SQL Database Memory Utilization > 90%"

  documentation {
    content   = "Cloud SQL Database Memory Utilization is >90%. Please increase memory capacity via the database tier (https://cloud.google.com/sql/docs/mysql/instance-settings)."
    mime_type = "text/markdown"
  }

  enabled               = "true"
  notification_channels = local.notification_channels
  project               = var.project_id

  user_labels = {
    severity = "warning"
  }
}

# Cloud SQL Database Memory Utilization > 95%
resource "google_monitoring_alert_policy" "cloud_sql_memory_utilization" {
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

      comparison      = "COMPARISON_GT"
      duration        = "0s"
      filter          = "metric.type=\"cloudsql.googleapis.com/database/memory/utilization\" resource.type=\"cloudsql_database\""
      threshold_value = "0.95"

      trigger {
        count   = "1"
        percent = "0"
      }
    }

    display_name = "Cloud SQL Database - Memory utilization [MEAN]"
  }

  display_name = "Cloud SQL Database Memory Utilization > 95%"

  documentation {
    content   = "Cloud SQL Database Memory Utilization is >95%. Please increase memory capacity via the database tier (https://cloud.google.com/sql/docs/mysql/instance-settings)."
    mime_type = "text/markdown"
  }

  enabled               = "true"
  notification_channels = local.notification_channels
  project               = var.project_id
}

# Cloud SQL Database Disk has < 20GiB Free
resource "google_monitoring_alert_policy" "cloud_sql_disk_utilization" {
  # In the absence of data, incident will auto-close in 7 days
  alert_strategy {
    auto_close = "604800s"
  }

  combiner = "OR"

  # Disk has less that 20GiB free
  conditions {
    # < 20GiB disk space free
    condition_monitoring_query_language {
      duration = "0s"
      query    = <<-EOT
        fetch cloudsql_database
        | { bytes: metric 'cloudsql.googleapis.com/database/disk/bytes_used'
          ; quota: metric 'cloudsql.googleapis.com/database/disk/quota'
          ; utilization: metric 'cloudsql.googleapis.com/database/disk/utilization' }
        | join
        | group_by 5m, [q_mean: mean(value.quota), b_mean: mean(value.bytes_used), u_mean: mean(value.utilization)]
        | every 5m
        | group_by [resource.database_id], [free_space: sub(mean(q_mean), mean(b_mean)), u: mean(u_mean)]
        | condition and(free_space < 20 'GiBy', u > 0.98)
      EOT
      trigger {
        count   = "1"
        percent = "0"
      }
    }

    display_name = "Cloud SQL Database - Disk free space and utilization [MEAN]"
  }

  display_name = "Cloud SQL Database Disk has < 20GiB Free and Utilization > 98%"

  documentation {
    content   = "Cloud SQL disk has less than 20GiB free space remaining. Please increase capacity. Note that autoresize should be enabled for the database. Ensure there is no issue with the autoresize process."
    mime_type = "text/markdown"
  }

  enabled               = "true"
  notification_channels = local.notification_channels
  project               = var.project_id
}


### Cloud SQL Proxy Alerts

# Cloud SQL Proxy Connection Failures
resource "google_monitoring_alert_policy" "cloudsqlconn_connection_failure" {
  # In the absence of data, incident will auto-close in 7 days
  alert_strategy {
    auto_close = "604800s"
  }

  combiner = "OR"

  # Connection failures are greater than 0
  conditions {
    condition_threshold {
      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_RATE"
      }

      comparison      = "COMPARISON_GT"
      duration        = "300s"
      filter          = "metric.type=\"prometheus.googleapis.com/cloudsqlconn_dial_failure_count/counter\" resource.type=\"prometheus_target\""
      threshold_value = "0"

      trigger {
        count   = "1"
        percent = "0"
      }
    }

    display_name = "Cloud SQL Proxy connections failing"
  }

  display_name = "Cloud SQL Proxy connections failing"

  documentation {
    content   = "Cloud SQL Proxy connections have been failing for at least 5 minutes.\n"
    mime_type = "text/markdown"
  }

  enabled               = "true"
  notification_channels = local.notification_channels
  project               = var.project_id
}


### KMS Alerts

resource "google_monitoring_alert_policy" "kms_read_request_alert" {
  # In the absence of data, incident will auto-close in 7 days

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

      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MEAN"
      }

      comparison = "COMPARISON_GT"
      duration   = "0s"
      filter     = "metric.type=\"serviceruntime.googleapis.com/quota/rate/net_usage\" resource.type=\"consumer_quota\" resource.label.\"service\"=\"cloudkms.googleapis.com\" metric.label.\"quota_metric\"=\"cloudkms.googleapis.com/read_requests\""
      // The threshold is 1500/min or 25/s
      threshold_value = "25"

      trigger {
        count   = "1"
        percent = "0"
      }
    }

    display_name = "KMS Read Requests Above Quota"
  }

  display_name = "KMS Read Requests quota usage"

  documentation {
    content   = "KMS Read Requests Above Quota, please see playbook for help."
    mime_type = "text/markdown"
  }

  enabled               = "true"
  notification_channels = local.notification_channels
  project               = var.project_id
}

resource "google_monitoring_alert_policy" "kms_crypto_request_alert" {
  # In the absence of data, incident will auto-close in 7 days
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

      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MEAN"
      }

      comparison = "COMPARISON_GT"
      duration   = "300s"
      filter     = "resource.type = \"consumer_quota\" AND resource.labels.service = \"cloudkms.googleapis.com\" AND metric.type = \"serviceruntime.googleapis.com/quota/rate/net_usage\" AND metric.labels.quota_metric = \"cloudkms.googleapis.com/crypto_requests\""
      // The threshold is 60,000/min or 1000/s
      threshold_value = "1000"

      trigger {
        count   = "1"
        percent = "0"
      }
    }

    display_name = "KMS Crypto Requests Rate quota usage"
  }

  display_name = "KMS Crypto Requests Rate Above Quota"

  documentation {
    content   = "KMS Crypto Requests Above Quota, please see playbook for help."
    mime_type = "text/markdown"
  }

  enabled               = "true"
  notification_channels = local.notification_channels
  project               = var.project_id
}

### Kubernetes Alerts

# Kubernetes Node Memory Allocatable Utilization > 90%
resource "google_monitoring_alert_policy" "k8s_container_memory_allocatable_utilization" {
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

      comparison      = "COMPARISON_GT"
      duration        = "0s"
      filter          = "metric.type=\"kubernetes.io/node/memory/allocatable_utilization\" resource.type=\"k8s_node\""
      threshold_value = "0.9"

      trigger {
        count   = "1"
        percent = "0"
      }
    }

    display_name = "Kubernetes Node - Memory Allocatable Utilization [MEAN]"
  }

  display_name = "Kubernetes Node Memory Allocatable Utilization > 90%"

  documentation {
    content   = "Kubernetes Node using >90% of allocatable memory. Please investigate possible memory leak."
    mime_type = "text/markdown"
  }

  enabled               = "true"
  notification_channels = local.notification_channels
  project               = var.project_id
}

# Kubernetes Node CPU Allocatable Utilization > 90%
resource "google_monitoring_alert_policy" "k8s_container_cpu_allocatable_utilization" {
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

      comparison      = "COMPARISON_GT"
      duration        = "0s"
      filter          = "metric.type=\"kubernetes.io/node/cpu/allocatable_utilization\" resource.type=\"k8s_node\""
      threshold_value = "0.9"

      trigger {
        count   = "1"
        percent = "0"
      }
    }

    display_name = "Kubernetes Node - CPU Allocatable Utilization [MEAN]"
  }

  display_name = "Kubernetes Node CPU Allocatable Utilization > 90%"

  documentation {
    content   = "Kubernetes Node using >90% of allocatable CPU. Please investigate running processes."
    mime_type = "text/markdown"
  }

  enabled               = "true"
  notification_channels = local.notification_channels
  project               = var.project_id
}

### Redis Alerts

# Redis Memory Usage > 90%
resource "google_monitoring_alert_policy" "redis_memory_usage" {
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

      comparison      = "COMPARISON_GT"
      duration        = "0s"
      filter          = "metric.type=\"redis.googleapis.com/stats/memory/usage_ratio\" resource.type=\"redis_instance\""
      threshold_value = "0.9"

      trigger {
        count   = "1"
        percent = "0"
      }
    }

    display_name = "Redis - Memory Usage [MEAN]"
  }

  display_name = "Redis Memory Usage > 90%"

  documentation {
    content   = "Redis using >90% of max memory. Playbook: https://github.com/sigstore/public-good-instance/blob/main/playbooks/alerting/alerts/redis-memory.md"
    mime_type = "text/markdown"
  }

  user_labels = {
    severity = "warning"
  }

  enabled               = "true"
  notification_channels = local.notification_channels
  project               = var.project_id
}

# Redis OOM
resource "google_monitoring_alert_policy" "redis_out_of_memory" {
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

      comparison      = "COMPARISON_GT"
      duration        = "0s"
      filter          = "metric.type=\"redis.googleapis.com/stats/memory/usage_ratio\" resource.type=\"redis_instance\""
      threshold_value = "0.99"

      trigger {
        count   = "1"
        percent = "0"
      }
    }

    display_name = "Redis - Out of Memory (99%) [MEAN]"
  }

  display_name = "Redis Out of Memory (99%)"

  documentation {
    content   = "Redis is out of memory. Playbook: https://github.com/sigstore/public-good-instance/blob/main/playbooks/alerting/alerts/redis-memory.md"
    mime_type = "text/markdown"
  }

  enabled               = "true"
  notification_channels = local.notification_channels
  project               = var.project_id
}
